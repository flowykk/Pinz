import SwiftUI
import PinzProfile
import PinzTrips
import PinzPins
import PinzFeed

public struct RootView<Content: View>: View {
    @Bindable var router: AppRouter
    let rootContent: Content

    public init(router: AppRouter, @ViewBuilder rootContent: () -> Content) {
        self.router = router
        self.rootContent = rootContent()
    }

    public var body: some View {
        NavigationStack(path: $router.path) {
            rootContent
                .navigationDestination(for: Route.self) { route in
                    destinationView(for: route).toolbar(.hidden)
                }
        }
        .environment(\.appRouter, router)
    }

    @ViewBuilder
    private func destinationView(for route: Route) -> some View {
        switch route {
        case let .profile(profileRoute):
            switch profileRoute {
            case .profile:
                ProfileView()
            case let .emailChange(email, action):
                EmailChangeView(email: email, onChangeSuccess: action.action)
            }
        }
    }
}
