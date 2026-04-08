import SwiftUI
import PhotosUI
import PinzUI
import PinzNetworking
import PinzBase
import PinzDomain

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

    var state: State = .info
    private(set) var isLoading: Bool = false
    private(set) var loadingStatus: LoadingStatus?

    var name: String = "name"
    var description: String? = "descr"
    var category: TripCategory = .active //.none
    var season: TripSeason = .spring // .none
    var medias: [LoadedMedia] = []

    private let networkService = NetworkService()
    private var router: AppRouting?

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

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            changeLoading(to: true, status: .uploadingMedia)

            let response = try await networkService.createTrip(
                name: name,
                description: description,
                category: category == .none ? nil : category.value,
                season: season == .none ? nil : season.value,
                filesToUpload: buildFilesToUpload()
            )

            try await uploadMedia(response: response)

            changeLoadingStatus(to: .formingPins)

            let groupingResponse = try await networkService.processMediaGrouping(
                tripId: response.tripId,
                media: buildMediaEntries(from: response)
            )

            let pins = RawPins(pins: groupingResponse.draftPins.map { $0.toRawPin() })
            dispatch(.navigate(.preprocessedPins(tripId: response.tripId, pins: pins)))
            changeLoading(to: false, status: nil)
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

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
