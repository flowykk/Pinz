import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain
import PinzUI

@MainActor
@Observable
final class InitialTripSetupViewModel {

    enum State: SegmentedItem {
        public var id: Self { self }

        case info
        case gallery

        public var content: SegmentedItemContent {
            switch self {
            case .info:
                .text(PinzBaseStrings.Common.Label.info)
            case .gallery:
                .text(PinzBaseStrings.Common.Label.gallery)
            }
        }
    }

    enum Route {
        case preprocessedPins(tripId: String, pins: RawPins)
        case back
    }

    enum Intent {
        case navigate(Route)
        case addMedias([PhotosPickerItem])
        case deleteMedia(UUID)
    }

    enum AsyncIntent {
        case `continue`
    }

    enum LoadingStatus {
        case uploadingMedia
        case formingPins

        var localizedValue: String {
            switch self {
            case .uploadingMedia: PinzBaseStrings.TripCreation.Loading.uploadingMedia
            case .formingPins: PinzBaseStrings.TripCreation.Loading.formingPins
            }
        }
    }

    static let tripNameMaxLength = 50
    static let tripDescriptionMaxLength = 5000

    private static let tripNameValidationPattern = #"^[A-Za-zА-Яа-яЁё0-9._]+$"#

    var state: State = .info
    private(set) var isLoading: Bool = false
    private(set) var loadingStatus: LoadingStatus?

    var name: String = ""
    var description: String?
    var category: TripCategory = .none
    var season: TripSeason = .none
    var medias: [LoadedMedia] = []

    private let networkService: NetworkServiceProtocol
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?

    init(networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.networkService = networkService
    }

    func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .preprocessedPins(let tripId, let pins):
                router?.navigateToTripCreationPreprocessedPins(tripId: tripId, pins: pins)
            case .back:
                router?.pop()
            }
        case let .addMedias(items):
            let placeholderIds = items.map { _ in UUID() }
            let placeholders = placeholderIds.map { LoadedMedia(id: $0, content: .loading) }
            medias.append(contentsOf: placeholders)

            Task {
                var hadLoadFailure = false
                await withTaskGroup(of: (UUID, LoadedMedia?).self) { group in
                    for (index, item) in items.enumerated() {
                        let id = placeholderIds[index]
                        group.addTask {
                            let loaded = await MediaLoader.shared.load(from: item, id: id)
                            return (id, loaded)
                        }
                    }

                    for await (id, loaded) in group {
                        if let loaded {
                            guard let idx = medias.firstIndex(where: { $0.id == id }) else { continue }
                            medias[idx] = loaded
                        } else {
                            medias.removeAll { $0.id == id }
                            hadLoadFailure = true
                        }
                    }
                }
                if hadLoadFailure {
                    showToast?(PinzBaseStrings.TripCreation.Toast.mediaLoadFailed)
                }
            }
        case let .deleteMedia(mediaId):
            medias.removeAll { $0.id == mediaId }
        }
    }

    func validateForContinue() -> String? {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedName.isEmpty {
            return PinzBaseStrings.TripCreation.Toast.nameEmpty
        }
        if trimmedName.count > Self.tripNameMaxLength {
            return PinzBaseStrings.TripCreation.Toast.nameTooLong(Self.tripNameMaxLength)
        }
        if trimmedName.range(of: Self.tripNameValidationPattern, options: .regularExpression) == nil {
            return PinzBaseStrings.TripCreation.Toast.nameInvalidChars
        }
        if category == .none {
            return PinzBaseStrings.TripCreation.Toast.categoryNotSelected
        }
        if season == .none {
            return PinzBaseStrings.TripCreation.Toast.seasonNotSelected
        }
        if medias.isEmpty {
            return PinzBaseStrings.TripCreation.Toast.mediasEmpty
        }
        if medias.contains(where: { if case .loading = $0.content { true } else { false } }) {
            return PinzBaseStrings.TripCreation.Toast.mediaLoadingInProgress
        }
        if let desc = description, desc.count > Self.tripDescriptionMaxLength {
            return PinzBaseStrings.TripCreation.Toast.descriptionTooLong(Self.tripDescriptionMaxLength)
        }
        return nil
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            guard !isLoading else { return }

            if let validationError = validateForContinue() {
                showToast?(validationError)
                return
            }

            changeLoading(to: true, status: .uploadingMedia)
            defer { changeLoading(to: false, status: nil) }

            let response: CreateTripDTO
            do {
                response = try await networkService.createTrip(
                    name: name,
                    description: description,
                    category: category == .none ? nil : category.value,
                    season: season == .none ? nil : season.value,
                    filesToUpload: buildFilesToUpload()
                )
            } catch {
                showToast?(PinzBaseStrings.TripCreation.Toast.createTripFailed)
                throw error
            }

            do {
                try await uploadMedia(response: response)
            } catch {
                if let mediaError = error as? MediaUploadError {
                    if case let .limitExceeded(kind, _, _) = mediaError {
                        showToast?(MediaUploadPreprocessor.localizedLimitMessage(for: kind))
                    }
                } else {
                    showToast?(PinzBaseStrings.TripCreation.Toast.uploadMediaFailed)
                }
                throw error
            }

            changeLoadingStatus(to: .formingPins)

            let groupingResponse: ProcessMediaGroupingDTO
            do {
                groupingResponse = try await networkService.processMediaGrouping(
                    tripId: response.tripId,
                    media: buildMediaEntries(from: response)
                )
            } catch {
                showToast?(PinzBaseStrings.TripCreation.Toast.groupingFailed)
                throw error
            }

            if groupingResponse.draftPins.isEmpty {
                showToast?(PinzBaseStrings.TripCreation.Toast.noPinsGenerated)
                return
            }

            let pins = RawPins(pins: groupingResponse.draftPins.map { $0.toRawPin() })
            dispatch(.navigate(.preprocessedPins(tripId: response.tripId, pins: pins)))
        }
    }

    private func buildFilesToUpload() -> [FileToUploadDTO] {
        medias.compactMap { media -> FileToUploadDTO? in
            guard case .loading = media.content else {
                return FileToUploadDTO(clientId: media.id.uuidString, contentType: media.uploadContentType)
            }
            return nil
        }
    }

    private func uploadMedia(response: CreateTripDTO) async throws {
        try await withThrowingTaskGroup(of: Void.self) { group in
            for uploadURL in response.uploadUrls {
                guard let media = medias.first(where: { $0.id.uuidString == uploadURL.clientId }) else {
                    continue
                }
                group.addTask { [weak self] in
                    guard let self else { return }
                    let prepared = try await self.prepareUpload(
                        for: media,
                        uploadURL: uploadURL.url,
                        context: "initial_trip_setup"
                    )
                    switch prepared.body {
                    case let .data(data):
                        try await self.networkService.uploadToS3(
                            url: uploadURL.url,
                            data: data,
                            contentType: prepared.contentType
                        )
                    case let .file(fileURL):
                        try await self.networkService.uploadToS3(
                            url: uploadURL.url,
                            fileURL: fileURL,
                            contentType: prepared.contentType
                        )
                    }
                }
            }
            try await group.waitForAll()
        }
    }

    private func prepareUpload(
        for media: LoadedMedia,
        uploadURL: String?,
        context: String
    ) async throws -> PreparedUpload {
        switch media.content {
        case .image(let image):
            return try await MediaUploadPreprocessor.shared.prepareImage(
                image,
                contentType: media.uploadContentType,
                uploadURL: uploadURL,
                context: context
            )
        case let .video(url: url, _):
            return try await MediaUploadPreprocessor.shared.prepareVideo(
                from: url,
                uploadURL: uploadURL,
                context: context
            )
        case .loading:
            throw MediaUploadError.invalidImageData
        }
    }

    private func buildMediaEntries(from response: CreateTripDTO) -> [MediaMetaEntryDTO] {
        response.uploadUrls.compactMap { uploadURL -> MediaMetaEntryDTO? in
            guard let media = medias.first(where: { $0.id.uuidString == uploadURL.clientId }) else { return nil }
            return MediaMetaEntryDTO(
                s3Key: uploadURL.s3Key,
                capturedAt: nil,
                latitude: media.coordinates?.latitude,
                longitude: media.coordinates?.longitude,
                mediaType: media.mediaType.rawValue,
                contentHash: nil
            )
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool, status: LoadingStatus?) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
            self.loadingStatus = status
        }
    }

    private func changeLoadingStatus(to status: LoadingStatus) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.loadingStatus = status
        }
    }
}
