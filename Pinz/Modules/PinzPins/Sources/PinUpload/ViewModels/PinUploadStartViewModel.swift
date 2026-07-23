import SwiftUI
import PhotosUI
import UIKit
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class PinUploadStartViewModel {

    enum Route {
        case processing(tripId: String, sessionId: String)
        case back
    }

    enum Intent {
        case navigate(Route)
        case addMedias([PhotosPickerItem])
        case deleteMedia(UUID)
    }

    enum AsyncIntent {
        case start
    }

    enum LoadingStatus {
        case uploading
        case committing
        case starting

        var localizedValue: String {
            switch self {
            case .uploading: PinzBaseStrings.PinUpload.Loading.uploading
            case .committing: PinzBaseStrings.PinUpload.Loading.committing
            case .starting: PinzBaseStrings.PinUpload.Loading.starting
            }
        }
    }

    let tripId: String
    let targetPinId: String?
    var medias: [LoadedMedia] = []
    private(set) var isLoading = false
    private(set) var loadingStatus: LoadingStatus?

    private var router: AppRouting?
    private let networkService: NetworkServiceProtocol

    init(tripId: String, targetPinId: String? = nil, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.targetPinId = targetPinId
        self.networkService = networkService
        if PinzLaunchArgs.testingPinUploadFakeMedia {
            medias = [Self.testingMedia()]
        }
    }

    // MARK: - dispatch

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case let .processing(tripId, sessionId):
                router?.navigateToPinUploadProcessing(tripId: tripId, sessionId: sessionId, targetPinId: targetPinId)
            }

        case let .addMedias(items):
            let placeholderIds = items.map { _ in UUID() }
            medias.append(contentsOf: placeholderIds.map { LoadedMedia(id: $0, content: .loading) })
            Task {
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
                        }
                    }
                }
            }

        case let .deleteMedia(mediaId):
            medias.removeAll { $0.id == mediaId }
        }
    }

    // MARK: - asyncDispatch

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .start:
            changeLoading(to: true, status: .uploading)
            defer { changeLoading(to: false, status: nil) }

            let filesToUpload: [FileToUploadDTO] = medias.compactMap { media in
                guard case .loading = media.content else {
                    return FileToUploadDTO(clientId: media.id.uuidString, contentType: media.uploadContentType)
                }
                return nil
            }

            let startResponse = try await networkService.pinUploadStart(
                tripId: tripId,
                targetPinId: targetPinId,
                filesToUpload: filesToUpload
            )

            // Persist session_id immediately, before any step that may fail —
            // GET /trips/{id} не отдаёт активную сессию, восстановление возможно только из локального хранилища.
            if let pinId = targetPinId {
                PinUploadAdditionSessionStorage.shared.save(
                    sessionId: startResponse.sessionId,
                    tripId: tripId,
                    pinId: pinId
                )
            } else {
                PinUploadSessionStorage.shared.save(
                    sessionId: startResponse.sessionId,
                    forTripId: tripId
                )
            }

            try await uploadToS3(uploadURLs: startResponse.uploadUrls)

            changeLoadingStatus(to: .committing)
            try await commitUploads(sessionId: startResponse.sessionId, uploadURLs: startResponse.uploadUrls)

            changeLoadingStatus(to: .starting)
            _ = try await networkService.pinUploadProcess(
                tripId: tripId,
                sessionId: startResponse.sessionId
            )

            dispatch(.navigate(.processing(tripId: tripId, sessionId: startResponse.sessionId)))
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    // MARK: - Helpers

    private func uploadToS3(uploadURLs: [UploadURLDTO]) async throws {
        try await withThrowingTaskGroup(of: Void.self) { group in
            for uploadURL in uploadURLs {
                guard let media = medias.first(where: { $0.id.uuidString == uploadURL.clientId }),
                      let data = await media.uploadData() else { continue }
                let contentType = media.uploadContentType
                group.addTask { [weak self] in
                    guard let self else { return }
                    try await networkService.uploadToS3(url: uploadURL.url, data: data, contentType: contentType)
                }
            }
            try await group.waitForAll()
        }
    }

    private func commitUploads(sessionId: String, uploadURLs: [UploadURLDTO]) async throws {
        try await withThrowingTaskGroup(of: Void.self) { group in
            for uploadURL in uploadURLs {
                guard let media = medias.first(where: { $0.id.uuidString == uploadURL.clientId }) else { continue }
                let s3Key = uploadURL.s3Key
                let mediaType = media.mediaType.rawValue
                let latitude = media.coordinates?.latitude
                let longitude = media.coordinates?.longitude
                group.addTask { [weak self] in
                    guard let self else { return }
                    _ = try await networkService.pinUploadCommitUpload(
                        tripId: tripId,
                        sessionId: sessionId,
                        s3Key: s3Key,
                        mediaType: mediaType,
                        capturedAtUnix: nil,
                        latitude: latitude,
                        longitude: longitude
                    )
                }
            }
            try await group.waitForAll()
        }
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

    private static func testingMedia() -> LoadedMedia {
        let renderer = UIGraphicsImageRenderer(size: CGSize(width: 8, height: 8))
        let image = renderer.image { context in
            UIColor.systemBlue.setFill()
            context.fill(CGRect(x: 0, y: 0, width: 8, height: 8))
        }
        return LoadedMedia(
            content: .image(image),
            imageFileData: image.pngData(),
            contentType: MediaType.image.contentType
        )
    }
}
