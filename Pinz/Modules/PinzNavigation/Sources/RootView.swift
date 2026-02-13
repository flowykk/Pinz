import SwiftUI
import PinzProfile
import PinzTrips
import PinzPins
import PinzFeed
import PinzMedias
import PinzDomain

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
            case .members:
                TripMembersView()
            }
        case let .tripInfo(tripInfoRoute):
            switch tripInfoRoute {
            case let .pinsList(trip):
                TripPinsListView(trip: trip)
            case let .selectablePinsList(trip):
                SelectablePinsListView(trip: trip)
            case let .postPreview(trip, selectedPins):
                PostPreviewView(trip: trip, selectedPins: selectedPins)
            }
        case let .profile(profileRoute):
            switch profileRoute {
            case let .emailChange(email, action):
                EmailChangeView(email: email, onChangeSuccess: action.action)
            case .statistics:
                StatisticsView()
            case .trips:
                TripsListView(trips: [Trip.stub(), Trip.stub()])
            case .wishlist:
                PlacesWishlistView()
            case .saved:
                SavedMapsView()
            case .notifications:
                NotificationsView()
            case .appearance:
                AppearanceView()
            }
        case let .pinInfo(pinInfoRoute):
            switch pinInfoRoute {
            case let .placeChange(pin, action):
                PinPlaceChangeView(pin: pin, onSave: action.action)
            }
        case let .media(mediaRoute):
            switch mediaRoute {
            case let .info(media):
                MediaInfoView(media: media)
            }
        }
    }
}
