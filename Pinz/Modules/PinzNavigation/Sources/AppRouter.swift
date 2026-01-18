import SwiftUI
import PinzDomain
import PinzBase

@MainActor @Observable
public final class AppRouter: AppRouting {
    public var path: [Route] = []

    public init() {}

    public func navigate(to route: Route) {
        path.append(route)
    }

    public func pop() {
        pop(by: 1)
    }

    public func pop(by count: Int) {
        guard count > 0, count <= path.count else { return }
        path.removeLast(count)
    }
}

// MARK: - Profile Routing

extension AppRouter {
    public func navigateToProfile() {
        navigate(to: .profile(.profile))
    }

    public func navigateToEmailChange(email: String, action: EmailChangeAction) {
        navigate(to: .profile(.emailChange(email: email, action: action)))
    }
}

// MARK: - Feed Routing

extension AppRouter {
    public func navigateToFeed() {
        // TODO: implement feed route
    }
}
