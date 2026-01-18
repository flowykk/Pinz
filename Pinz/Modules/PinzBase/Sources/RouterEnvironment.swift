import SwiftUI
import PinzDomain

public protocol AppRouting {
    func navigateToProfile()
    func navigateToEmailChange(email: String, action: EmailChangeAction)

    func navigateToFeed()
    
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
