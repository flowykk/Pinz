import SwiftUI
import PinzNetworking
import PinzNavigation

@Observable
public class ProfileViewModel {

    public enum State {
        case `default`
        case editing
    }

    public enum Intent {
        case back
        case addPerson
        case changeState
    }

    var state: State = .default
    var navigator = Navigator<ProfileDestination>()
    var nickname: String
    var email: String

    private let networkService = NetworkService()

    public init(nickname: String, email: String) {
        self.nickname = nickname
        self.email = email
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .back:
            navigator.back()
        case .addPerson:
            navigator.navigate(to: .addPerson)
        case .changeState:
            switch state {
            case .default: changeState(to: .editing)
            case .editing: changeState(to: .default)
            }
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
