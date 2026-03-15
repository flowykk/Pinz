import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class WishlistCreationViewModel {

    enum State {
        case name
        case description
        case photo
    }

    enum Route {
        case back
    }

    enum Intent {
        case `continue`
        case selectPhoto(PhotosPickerItem)
        case navigate(Route)
    }

    var state: State = .name
    var image: UIImage?
    var name: String = ""
    var description: String = ""

    private let onCreated: (WishlistElement) -> Void
    private let networkService = NetworkService()
    private var router: AppRouting?

    var isCompleteButtonDisabled: Bool {
        switch state {
        case .name:
            return name.isEmpty
        case .description:
            return description.isEmpty
        case .photo:
            return image == nil
        }
    }

    init(onCreated: @escaping (WishlistElement) -> Void) {
        self.onCreated = onCreated
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .continue:
            switch state {
            case .name:        changeState(to: .description)
            case .description: changeState(to: .photo)
            case .photo:
                guard let image else { return }
                onCreated(WishlistElement(image: image, title: name, description: description))
                router?.pop()
            }
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

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
