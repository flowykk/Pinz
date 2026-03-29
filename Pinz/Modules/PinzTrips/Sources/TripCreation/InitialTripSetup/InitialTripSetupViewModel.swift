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
                .text("Информация")
            case .gallery:
                .text("Гелерея")
            }
        }
    }

    enum Route {
        case preprocessedPins
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

    var state: State = .info
    private(set) var isLoading: Bool = false

    var name: String = ""
    var description: String?
    var category: TripCategory = .none
    var season: TripSeason = .none
    var medias: [LoadedMedia] = []

    private let networkService = NetworkService()
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .preprocessedPins:
                router?.navigateToTripCreationPreprocessedPins()
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
            changeLoading(to: true)
            try await Task.sleep(nanoseconds: 1_000_000_000)
            dispatch(.navigate(.preprocessedPins))
            changeLoading(to: false)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
