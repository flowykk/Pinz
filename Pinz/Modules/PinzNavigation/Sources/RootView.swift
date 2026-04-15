import SwiftUI
import PinzAuthentication
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
                .toolbar(.hidden)
                .navigationDestination(for: Route.self) { route in
                    destinationView(for: route).toolbar(.hidden)
                }
        }
        .environment(\.appRouter, router)
    }

    @ViewBuilder
    private func destinationView(for route: Route) -> some View {
        switch route {
        case .main:
            TripView(trips: Trip.stubs())
                .navigationBarBackButtonHidden(true)
        case let .trip(tripRoute):
            switch tripRoute {
            case let .info(trip):
                TripInfoView(trip: trip, onTripUpdated: consumeTripInfoUpdateHandler())
            case let .profile(user):
                ProfileView(user: user)
            case let .pinInfo(pin, updateAction):
                PinInfoView(pin: pin, updateAction: updateAction)
            case .pinCreation:
                PinCreationView()
            case .members:
                TripMembersView()
            case .feed:
                FeedView()
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
        case let .tripCreation(tripCreationRoute):
            switch tripCreationRoute {
            case .initial:
                InitialTripSetupView()
            case .preprocessed(let tripId, let pins):
                PreprocessedRawPinsView(tripId: tripId, pins: pins)
            case .final(let tripId, let pins):
                ReviewTripCreationView(tripId: tripId, pins: pins)
            }
        case let .profile(profileRoute):
            switch profileRoute {
            case let .emailChange(email, userId, action):
                EmailChangeView(email: email, userId: userId, onChangeSuccess: action.action)
            case .statistics:
                StatisticsView()
            case .trips:
                TripsListView()
            case .wishlist:
                WishlistView()
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
            case let .localInfo(media):
                MediaInfoView(localMedia: media)
            }
        case let .wishlist(wishlistRoute):
            switch wishlistRoute {
            case let .element(element):
                WishlistElementView(element: element)
            case let .creation(action):
                WishlistElementCreationView(onCreated: action.action)
            }
        }
    }

    private func consumeTripInfoUpdateHandler() -> (() -> Void)? {
        router.consumeTripInfoUpdateHandler()
    }
}
