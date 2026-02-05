import SwiftUI
import PinzNetworking
import PinzDomain
import PinzUI
import PinzBase

@Observable
public class ProfileViewModel {

    public enum State {
        case `default`
        case editing
    }

    public enum Route {
        case emailChange

        case statistics

        case trips
        case wishlist
        case saved

        case notifications
        case appearance

        case back
    }

    public enum Intent {
        case changeState
        case setImage(UIImage?)
        case navigate(Route)
    }

    var state: State = .default

    var user: User
    var userImage: UIImage = PinzUIAsset.avatar.image
    private let networkService = NetworkService()
    private var router: AppRouting?

    var imageBinding: Binding<UIImage?> {
        Binding {
            self.userImage
        } set: { newImage in
            guard let newImage else {
                return
            }
            self.userImage = newImage
        }
    }

    public init(user: User) {
        self.user = user
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .changeState:
            switch state {
            case .default: changeState(to: .editing)
            case .editing: changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                userImage = newImage
            }
        case let .navigate(route):
            switch route {
            case .emailChange:
                let action = EmailChangeAction { [weak self] newEmail in
                    self?.user.email = newEmail
                    self?.router?.pop()
                }
                router?.navigateToEmailChange(email: user.email, action: action)
            case .statistics:
                router?.navigateToStatistics()
            case .trips:
                router?.navigateToTrips()
            case .wishlist:
                router?.navigateToPlacesWishlist()
            case .saved:
                router?.navigateToSavedMaps()
            case .notifications:
                router?.navigateToNotifications()
            case .appearance:
                router?.navigateToAppearance()
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
