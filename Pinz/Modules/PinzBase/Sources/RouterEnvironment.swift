import SwiftUI
import PinzDomain

public protocol AppRouting {
    func navigateToTripInfo(trip: Trip)
    func navigateToProfile(user: User)
    func navigateToPinInfo(pin: Pin)

    func navigateToEmailChange(email: String, action: EmailChangeAction)
    func navigateToAppearance()

    func navigateToFeed()

    func navigateToPinPlaceChange(pin: Pin, action: PlaceSaveAction)

    func pop()
    func pop(by count: Int)
}

private struct AppRouterKey: EnvironmentKey {
    static let defaultValue: AppRouting? = nil
}

public extension EnvironmentValues {
    var appRouter: AppRouting? {
        get { self[AppRouterKey.self] }
        set { self[AppRouterKey.self] = newValue }
    }
}
