import SwiftUI
import PinzNetworking

@Observable
class EmailChangeViewModel {

    public enum State {
        case firstCode
        case email
        case secondCode
    }

    public enum Intent {
        case `continue`
    }

    var successAction: (String) -> Void
    var dismiss: DismissAction? = nil
    var state: State = .firstCode

    var code: [String] = Array(repeating: "", count: 4)
    var email: String

    private let networkService = NetworkService()

    public init(
        email: String,
        successAction: @escaping (String) -> Void
    ) {
        self.email = email
        self.successAction = successAction
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .continue:
            switch state {
            case .firstCode:
                changeState(to: .email)
                code = Array(repeating: "", count: 4)
            case .email:
                changeState(to: .secondCode)
            case .secondCode:
                successAction(email)
                dismiss?()
            }
        }
    }

    func setDismiss(_ dismiss: DismissAction) {
        self.dismiss = dismiss
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
