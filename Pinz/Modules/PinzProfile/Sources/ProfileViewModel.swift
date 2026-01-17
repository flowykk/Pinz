import SwiftUI
import PinzNetworking
import PinzNavigation
import PinzDomain
import PinzUI

@Observable
public class ProfileViewModel {

    public enum State {
        case `default`
        case editing
    }

    public enum Intent {
        case back
        case changeState

        case setImage(UIImage?)
        case setEmail(String)
    }

    var state: State = .default

    var user: User
    var userImage: UIImage = PinzUIAsset.avatar.image
    var navigator = Navigator<ProfileDestination>()
    private let networkService = NetworkService()

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
        case .back:
            navigator.back()
        case .changeState:
            switch state {
            case .default: changeState(to: .editing)
            case .editing: changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                userImage = newImage
            }
        case let .setEmail(newEmail):
            user.email = newEmail
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
