import SwiftUI
import PinzBase
import PinzNetworking

@MainActor
@Observable
class EmailChangeViewModel {

    public enum Route {
        case back
    }

    public enum State {
        case email
        case code
    }

    public enum Intent {
        case navigate(Route)
    }

    var successAction: (String) -> Void
    var state: State = .email
    var isLoading = false
    var email: String
    var userId: String?
    var code: [String] = Array(repeating: "", count: 4)

    private var showToast: ((String) -> Void)?
    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?

    var isNextButtonDisabled: Bool {
        switch state {
        case .email:
            return normalizedEmail.isEmpty
        case .code:
            return verificationCode.count < 4
        }
    }

    var nextButtonTitle: String {
        switch state {
        case .email:
            PinzBaseStrings.EmailChange.Button.receiveCode
        case .code:
            PinzBaseStrings.Common.Button.done
        }
    }

    private var verificationCode: String {
        code.joined()
    }

    private var normalizedEmail: String {
        email.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var requestUserId: String? {
        let trimmed = userId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }

    public init(
        email: String,
        userId: String? = nil,
        networkService: (any NetworkServiceProtocol)? = nil,
        successAction: @escaping (String) -> Void
    ) {
        self.email = email
        self.userId = userId
        self.networkService = networkService ?? NetworkService.shared
        self.successAction = successAction
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    public func continueTapped() async {
        guard !isLoading else {
            return
        }

        switch state {
        case .email:
            await requestCode()
        case .code:
            await confirmCode()
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    public func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    private func requestCode() async {
        let targetEmail = normalizedEmail
        guard !targetEmail.isEmpty else { return }

        guard targetEmail.isValidEmail else {
            showToast?(PinzBaseStrings.EmailChange.Toast.invalidEmail)
            return
        }

        setLoading(true)
        defer { setLoading(false) }

        do {
            _ = try await networkService.changeEmail(userId: requestUserId, newEmail: targetEmail)
            email = targetEmail
            code = Array(repeating: "", count: 4)
            changeState(to: .code)
        } catch {
            print("[EmailChange] Failed to request email change: \(error)")
            showToast?(PinzBaseStrings.EmailChange.Toast.codeSendFailed)
        }
    }

    private func confirmCode() async {
        let codeToConfirm = verificationCode
        guard codeToConfirm.count == 4 else {
            return
        }

        setLoading(true)
        defer {
            setLoading(false)
        }

        do {
            let response = try await networkService.confirmEmailChange(verificationCode: codeToConfirm)
            successAction(response.email ?? email)
        } catch {
            print("[EmailChange] Failed to confirm email change: \(error)")
            showToast?(PinzBaseStrings.EmailChange.Toast.confirmFailed)
        }
    }

    private func setLoading(_ loading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            isLoading = loading
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }

}
