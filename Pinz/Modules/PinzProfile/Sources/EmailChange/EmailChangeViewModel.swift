import SwiftUI
import PinzNetworking
import PinzBase

@Observable
class EmailChangeViewModel {

    public enum State {
        case firstCode
        case email
        case secondCode
    }

    public enum Route {
        case back
    }

    public enum Intent {
        case navigate(Route)
        case `continue`
    }

    var successAction: (String) -> Void
    var state: State = .firstCode

    var code: [String] = Array(repeating: "", count: 4)
    var email: String

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    public init(
        email: String,
        successAction: @escaping (String) -> Void
    ) {
        self.email = email
        self.successAction = successAction
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case .continue:
            switch state {
            case .firstCode:
                changeState(to: .email)
                code = Array(repeating: "", count: 4)
            case .email:
                changeState(to: .secondCode)
            case .secondCode:
                successAction(email)
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
