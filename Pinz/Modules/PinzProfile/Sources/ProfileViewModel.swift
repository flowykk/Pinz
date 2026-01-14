import SwiftUI
import PinzNetworking

@Observable
public class ProfileViewModel {

    public enum State {
        case `default`
        case editing
    }

    public enum Intent {
        case back
        case addPerson
        case changeState(State)
    }

    public var state: State = .default
    public var nickname: String
    public var email: String

    private let networkService = NetworkService()

    public init(nickname: String, email: String) {
        self.nickname = nickname
        self.email = email
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .back:
            print("back")
        case .addPerson:
            print("add person")
        case let .changeState(state):
            self.state = state
        }
    }
}
