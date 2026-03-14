import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase

@MainActor
@Observable
final class WishlistCreationViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case complete
        case selectPhoto(PhotosPickerItem)
        case navigate(Route)
    }

    var image: UIImage?
    var name: String = ""
    var description: String = ""

    private let networkService = NetworkService()
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case .complete:
            router?.pop()
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else { return }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        image = uiImage
                    }
                }
            }
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
