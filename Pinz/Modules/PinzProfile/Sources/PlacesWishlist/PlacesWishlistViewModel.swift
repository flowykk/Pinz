import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PlacesWishlistViewModel {

    enum Route {
        case wishlistElementCreation
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    var wishlist: [WishlistElement]

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(wishlist: [WishlistElement] = WishlistElement.stubs) {
        self.wishlist = wishlist
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .wishlistElementCreation:
                router?.navigateToWishlistElementCreation(action: WishlistCreationAction { [weak self] element in
                    self?.wishlist.append(element)
                })
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
