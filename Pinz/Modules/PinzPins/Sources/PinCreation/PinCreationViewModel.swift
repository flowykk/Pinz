import SwiftUI
import MapKit
import PhotosUI
import PinzUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class PinCreationViewModel {

    public enum State: SegmentedItem {
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
        case back
    }

    enum Intent {
        case addTag(MediaTag)
        case deleteTag(MediaTag)
        case addMedias([PhotosPickerItem])

        case navigate(Route)
    }

    var state: State = .info

    var name: String = ""
    var description: String?
    var category: PinCategory = .custom()
    var startDate: Date?
    var endDate: Date?
    var medias: [LoadedMedia] = []
    var tags: [MediaTag] = []

    private let networkService = NetworkService()
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .addTag(tag):
            tags.append(tag)
        case let .deleteTag(tag):
            tags.removeAll { $0.tag == tag.tag }
        case let .addMedias(items):
            let placeholderIds = items.map { _ in UUID() }
            let placeholders = placeholderIds.map { LoadedMedia(id: $0, content: .loading) }
            medias.append(contentsOf: placeholders)

            Task {
                for (index, item) in items.enumerated() {
                    let id = placeholderIds[index]
                    if let loaded = await MediaLoader.shared.load(from: item, id: id) {
                        guard let idx = medias.firstIndex(where: { $0.id == id }) else { continue }
                        medias[idx] = loaded
                    } else {
                        medias.removeAll { $0.id == id }
                    }
                }
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
