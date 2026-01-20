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

// MARK: - Trip Routing

extension AppRouter {
    public func navigateToTripInfo(trip: Trip) {
        navigate(to: .trip(.info(trip: trip)))
    }

    public func navigateToProfile(user: User) {
        navigate(to: .trip(.profile(user: user)))
    }
}

// MARK: - Profile Routing

extension AppRouter {
    public func navigateToEmailChange(email: String, action: EmailChangeAction) {
        navigate(to: .profile(.emailChange(email: email, action: action)))
    }

    public func navigateToAppearance() {
        navigate(to: .profile(.appearance))
    }
}

// MARK: - Feed Routing

extension AppRouter {
    public func navigateToFeed() {
        // TODO: implement feed route
    }
}
