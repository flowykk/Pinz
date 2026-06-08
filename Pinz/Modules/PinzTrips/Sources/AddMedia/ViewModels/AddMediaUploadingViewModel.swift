import SwiftUI
import PhotosUI
import Foundation
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class AddMediaUploadingViewModel {

    enum Route {
        case grouping(tripId: String, sessionId: String)
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case processGrouping
        case addMore([PhotosPickerItem])
        case cancel
    }

    let tripId: String
    let sessionId: String
    var uploadedMediaEntries: [AddMediaSessionMediaEntryDTO] = []
    private(set) var isLoading = false

    private var wsTask: Task<Void, Never>?
    private var wsClient = AddMediaWebSocketClient()
    private var router: AppRouting?
    private let networkService: NetworkServiceProtocol

    init(tripId: String, sessionId: String, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.networkService = networkService
    }

    // MARK: - dispatch

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case let .grouping(tripId, sessionId):
                router?.navigateToAddMediaGrouping(tripId: tripId, sessionId: sessionId)
            }
        }
    }

    // MARK: - asyncDispatch

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .processGrouping:
            isLoading = true
            defer { isLoading = false }
            _ = try await networkService.addMediaProcessGrouping(tripId: tripId, sessionId: sessionId, addMore: false)
            dispatch(.navigate(.grouping(tripId: tripId, sessionId: sessionId)))

        case let .addMore(items):
            let loadedMedias: [LoadedMedia] = await withTaskGroup(of: LoadedMedia?.self) { group in
                for item in items {
                    let id = UUID()
                    group.addTask { await MediaLoader.shared.load(from: item, id: id) }
                }
                var results: [LoadedMedia] = []
                for await loaded in group {
                    if let loaded { results.append(loaded) }
                }
                return results
            }

            let filesToUpload: [FileToUploadDTO] = loadedMedias.compactMap { media in
                guard case .loading = media.content else {
                    return FileToUploadDTO(clientId: media.id.uuidString, contentType: media.uploadContentType)
                }
                return nil
            }

            let uploadURLs = try await networkService.addMediaRequestUploadUrls(
                tripId: tripId,
                sessionId: sessionId,
                filesToUpload: filesToUpload
            )

            try await withThrowingTaskGroup(of: Void.self) { group in
                for uploadURL in uploadURLs {
                    guard let media = loadedMedias.first(where: { $0.id.uuidString == uploadURL.clientId }),
                          let data = await media.uploadData() else { continue }
                    let contentType = media.uploadContentType
                    group.addTask { [weak self] in
                        guard let self else { return }
                        try await networkService.uploadToS3(url: uploadURL.url, data: data, contentType: contentType)
                    }
                }
                try await group.waitForAll()
            }

            try await withThrowingTaskGroup(of: Void.self) { group in
                for uploadURL in uploadURLs {
                    guard let media = loadedMedias.first(where: { $0.id.uuidString == uploadURL.clientId }) else { continue }
                    let s3Key = uploadURL.s3Key
                    let mediaType = media.mediaType.rawValue
                    let latitude = media.coordinates?.latitude
                    let longitude = media.coordinates?.longitude
                    let capturedAt = media.capturedAt
                    group.addTask { [weak self] in
                        guard let self else { return }
                        _ = try await networkService.addMediaCommitUpload(
                            tripId: tripId,
                            sessionId: sessionId,
                            s3Key: s3Key,
                            mediaType: mediaType,
                            capturedAt: capturedAt,
                            latitude: latitude,
                            longitude: longitude
                        )
                    }
                }
                try await group.waitForAll()
            }

            let refreshed = try await networkService.addMediaGetSessionMedia(tripId: tripId, sessionId: sessionId)
            uploadedMediaEntries = refreshed.media

        case .cancel:
            try await networkService.addMediaCancel(tripId: tripId, sessionId: sessionId)
            router?.popToRoot()
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
        loadSessionMediaAndStartWS()
    }

    // MARK: - Private

    private func loadSessionMediaAndStartWS() {
        Task {
            do {
                let response = try await networkService.addMediaGetSessionMedia(tripId: tripId, sessionId: sessionId)
                uploadedMediaEntries = response.media
            } catch {
                print("[AddMediaUploadingViewModel] Failed to load session media: \(error)")
            }
            startWSListener()
        }
    }

    private func startWSListener() {
        wsTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId) {
                switch event {
                case let .addMediaProgress(mediaId, mediaUrl, mediaType, actorUserId, _):
                    guard !uploadedMediaEntries.contains(where: { $0.mediaId == mediaId }) else { continue }
                    uploadedMediaEntries.append(AddMediaSessionMediaEntryDTO(
                        mediaId: mediaId,
                        url: mediaUrl,
                        type: mediaType,
                        actorUserId: actorUserId,
                        uploadedAt: ISO8601DateFormatter().string(from: Date())
                    ))
                case let .tripStatusChanged(status) where status.uppercased() == "ADD_MEDIA_GROUPING_REVIEW":
                    dispatch(.navigate(.grouping(tripId: tripId, sessionId: sessionId)))
                default:
                    break
                }
            }
        }
    }

    deinit {
        wsTask?.cancel()
        wsClient.disconnect()
    }
}
