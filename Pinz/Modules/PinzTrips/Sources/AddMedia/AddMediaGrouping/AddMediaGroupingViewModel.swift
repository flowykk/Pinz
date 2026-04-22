import SwiftUI
import PinzBase
import PinzNetworking
import PinzDomain

@MainActor @Observable
final class AddMediaGroupingViewModel {
    enum FlowStatus {
        case idle
        case uploading
        case grouping
        case ready
        case failed
    }

    enum LoadingStatus {
        case uploading
        case grouping
        case applying

        var localizedValue: String {
            switch self {
            case .uploading:
                PinzBaseStrings.TripCreation.Loading.uploadingMedia
            case .grouping:
                PinzBaseStrings.AddMedia.Loading.grouping
            case .applying:
                PinzBaseStrings.AddMedia.Loading.applying
            }
        }
    }

    struct PreparedUploadItem {
        let url: String
        let data: Data
        let contentType: String
        let s3Key: String
        let mediaType: String
        let latitude: Double?
        let longitude: Double?
    }

    let tripId: String
    private(set) var isLoading = false
    private(set) var loadingStatus: LoadingStatus?
    private(set) var isReady = false
    private(set) var draftPins: [RawPin] = []
    private(set) var existingMediaIds: Set<String> = []
    private(set) var existingPinsPreview: [RawPin] = []
    private(set) var hasFailed = false
    private(set) var deletedMediaIds: Set<String> = []
    private(set) var flowStatus: FlowStatus = .idle

    let session: AddMediaStartDTO
    private let loadedMedia: [LoadedMedia]
    private let networkService = NetworkService.shared

    init(
        tripId: String,
        session: AddMediaStartDTO,
        loadedMedia: [LoadedMedia]
    ) {
        self.tripId = tripId
        self.session = session
        self.loadedMedia = loadedMedia
    }

    enum Intent {
        case deleteMedia(RawPinMedia, fromPin: String)
        case moveMedia(RawPinMedia, fromPinId: String, toPinId: String)
    }

    enum AsyncIntent {
        case startGrouping
        case applyGroupsAndProcess
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .deleteMedia(media, pinId):
            guard !existingMediaIds.contains(media.id) else {
                return
            }
            withAnimation(.easeInOut(duration: 0.3)) {
                if let pinIndex = draftPins.firstIndex(where: { $0.id == pinId }) {
                    draftPins[pinIndex].medias.removeAll { $0.id == media.id }
                    deletedMediaIds.insert(media.id)
                }
            }
        case let .moveMedia(media, fromPinId, toPinId):
            guard !existingMediaIds.contains(media.id),
                  fromPinId != toPinId,
                  let fromPinIndex = draftPinIndex(for: fromPinId),
                  let toPinIndex = draftPinIndex(for: toPinId) else {
                return
            }

            withAnimation(.easeInOut(duration: 0.3)) {
                draftPins[fromPinIndex].medias.removeAll { $0.id == media.id }
                draftPins[toPinIndex].medias.append(media)
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .startGrouping:
            try await startGrouping()
        case .applyGroupsAndProcess:
            try await applyGroupsAndProcess()
        }
    }

    private func startGrouping() async throws {
        changeLoading(to: true, status: .uploading)
        hasFailed = false
        isReady = false
        flowStatus = .uploading

        do {
            let preparedItems = try await prepareUploadItems()
            try await uploadMedia(preparedItems: preparedItems)
            changeLoading(to: true, status: .grouping)
            flowStatus = .grouping

            let mediaEntries = preparedItems.map {
                MediaMetaEntryDTO(
                    s3Key: $0.s3Key,
                    capturedAt: nil,
                    latitude: $0.latitude,
                    longitude: $0.longitude,
                    mediaType: $0.mediaType,
                    contentHash: nil
                )
            }

            let response = try await networkService.addMediaProcessGrouping(
                tripId: tripId,
                sessionId: session.sessionId,
                media: mediaEntries
            )

            existingMediaIds = Set(response.existingMediaIds ?? [])
            draftPins = sanitize(responseDraftPins: response.draftPins.map { $0.toRawPin() })
            existingPinsPreview = await loadExistingPinsPreview()
            isReady = true
            flowStatus = .ready
            changeLoading(to: false, status: nil)
        } catch {
            changeLoading(to: false, status: nil)
            hasFailed = true
            flowStatus = .failed
            if isRestartNeeded(for: error) {
                throw GroupingError.restartRequired
            }
            throw error
        }
    }

    private func prepareUploadItems() async throws -> [PreparedUploadItem] {
        let mediaByClientId = Dictionary(uniqueKeysWithValues: loadedMedia.map { ($0.id.uuidString, $0) })
        var preparedItems: [PreparedUploadItem] = []
        preparedItems.reserveCapacity(session.uploadUrls.count)

        for uploadUrl in session.uploadUrls {
            guard
                !uploadUrl.url.isEmpty,
                !uploadUrl.s3Key.isEmpty,
                let media = mediaByClientId[uploadUrl.clientId]
            else {
                continue
            }

            guard let data = await media.uploadData() else { continue }

            preparedItems.append(
                PreparedUploadItem(
                    url: uploadUrl.url,
                    data: data,
                    contentType: media.uploadContentType,
                    s3Key: uploadUrl.s3Key,
                    mediaType: media.mediaType.rawValue,
                    latitude: media.coordinates?.latitude,
                    longitude: media.coordinates?.longitude
                )
            )
        }

        guard !preparedItems.isEmpty else {
            throw GroupingError.emptyPayload
        }

        return preparedItems
    }

    private func uploadMedia(preparedItems: [PreparedUploadItem]) async throws {
        try await withThrowingTaskGroup(of: Void.self) { group in
            for item in preparedItems {
                group.addTask { [weak self] in
                    guard let self else { return }
                    try await networkService.uploadToS3(
                        url: item.url,
                        data: item.data,
                        contentType: item.contentType
                    )
                }
            }
            try await group.waitForAll()
        }
    }

    private func loadExistingPinsPreview() async -> [RawPin] {
        do {
            let response = try await networkService.getTrip(id: tripId)
            return response.pins.map { $0.toRawPin() }
        } catch {
            return []
        }
    }

    private func applyGroupsAndProcess() async throws {
        guard canProceed else {
            return
        }

        changeLoading(to: true, status: .applying)
        hasFailed = false
        flowStatus = .ready

        do {
            let draftPinsRequest = buildDraftPinsRequest()
            _ = try await networkService.addMediaApplyGroupsAndProcess(
                tripId: tripId,
                sessionId: session.sessionId,
                draftPins: draftPinsRequest,
                deletedMediaIds: Array(deletedMediaIds)
            )
            flowStatus = .ready
            changeLoading(to: false, status: nil)
        } catch {
            changeLoading(to: false, status: nil)
            setFailedState()
            throw error
        }
    }

    var canProceed: Bool {
        !draftPins.isEmpty
    }

    private func changeLoading(to isLoading: Bool, status: LoadingStatus?) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
            self.loadingStatus = status
        }
    }

    private func isRestartNeeded(for error: Error) -> Bool {
        if let httpError = error as? HTTPError {
            return httpError == .conflict || httpError == .preconditionFailed
        }
        return false
    }

    private func sanitize(responseDraftPins: [RawPin]) -> [RawPin] {
        responseDraftPins.compactMap { pin in
            var sanitizedPin = pin
            sanitizedPin.medias.removeAll { existingMediaIds.contains($0.id) }
            return sanitizedPin.medias.isEmpty ? nil : sanitizedPin
        }
    }

    func setFailedState() {
        hasFailed = true
        flowStatus = .failed
    }

    private func buildDraftPinsRequest() -> [DraftPinInputDTO] {
        draftPins.compactMap { pin in
            let mediaIds = pin.medias
                .map(\.id)
                .filter { !existingMediaIds.contains($0) }

            guard !mediaIds.isEmpty else {
                return nil
            }
            return DraftPinInputDTO(draftPinId: pin.id, mediaIds: mediaIds)
        }
    }

    private func draftPinIndex(for id: String) -> Int? {
        draftPins.firstIndex(where: { $0.id == id })
    }
}

extension AddMediaGroupingViewModel {
    enum GroupingError: Error {
        case emptyPayload
        case restartRequired
    }
}
