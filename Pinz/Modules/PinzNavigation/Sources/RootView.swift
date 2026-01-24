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
        case let .trip(tripRoute):
            switch tripRoute {
            case let .info(trip):
                TripInfoView(trip: trip)
            case let .profile(user):
                ProfileView(user: user)
            case let .pinInfo(pin):
                PinInfoView(pin: pin)
            }
        case let .profile(profileRoute):
            switch profileRoute {
            case let .emailChange(email, action):
                EmailChangeView(email: email, onChangeSuccess: action.action)
            case .appearance:
                AppearanceView()
            }
        }
    }
}
