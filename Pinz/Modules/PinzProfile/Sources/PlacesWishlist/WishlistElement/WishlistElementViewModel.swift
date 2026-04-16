import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class WishlistElementViewModel {

    enum State {
        case `default`
        case editing
    }

    enum Route {
        case back
    }

    enum Intent {
        case selectPhoto(PhotosPickerItem)
        case edit
        case endEdit

        case navigate(Route)
    }

    var element: WishlistElement
    var state: State = .default

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    init(element: WishlistElement) {
        self.element = element
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else { return }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        element.image = uiImage
                    }
                }
            }
        case .edit:
            changeState(to: .editing)
        case .endEdit:
            changeState(to: .default)
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
