import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class WishlistViewModel {

    enum Route {
        case wishlistElement(DesiredPlace)
        case wishlistElementCreation
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    var wishlist: [DesiredPlace]
    var isLoading = false

    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?

    init(wishlist: [DesiredPlace] = [], networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.wishlist = wishlist
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .wishlistElement(element):
                router?.navigateToWishlistElement(element: element)
            case .wishlistElementCreation:
                router?.navigateToWishlistElementCreation(action: WishlistCreationAction { [weak self] element in
                    self?.wishlist.append(element)
                })
            case .back:
                router?.pop()
            }
        }
    }

    func loadWishlist() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let places = try await networkService.getDesiredPlaces()
            wishlist = places.map { $0.toDesiredPlace() }
        } catch {}
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
