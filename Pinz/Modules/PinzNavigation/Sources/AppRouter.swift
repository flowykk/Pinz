import SwiftUI
import PinzDomain
import PinzBase

@MainActor @Observable
public final class AppRouter: AppRouting {
    public var path: [Route]

    public init(initialPath: [Route] = []) {
        self.path = initialPath
    }

    public func navigateToMain() {
        path = [.main]
    }

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

    public func navigateToPinInfo(pin: Pin) {
        navigate(to: .trip(.pinInfo(pin: pin)))
    }

    public func navigateToPinCreation() {
        navigate(to: .trip(.pinCreation))
    }

    public func navigateToTripMembers() {
        navigate(to: .trip(.members))
    }

    public func navigateToFeed() {
        navigate(to: .trip(.feed))
    }
}

// MARK: - TripInfo Routing

extension AppRouter {
    public func navigateToPinsList(trip: Trip) {
        navigate(to: .tripInfo(.pinsList(trip: trip)))
    }

    public func navigateToSelectablePinsList(trip: Trip) {
        navigate(to: .tripInfo(.selectablePinsList(trip: trip)))
    }

    public func navigateToPostPreview(trip: Trip, selectedPins: [Pin]) {
        navigate(to: .tripInfo(.postPreview(trip: trip, selectedPins: selectedPins)))
    }
}

// MARK: - TripCreation Routing
extension AppRouter {
    public func navigateToTripCreationInitial() {
        navigate(to: .tripCreation(.initial))
    }

    public func navigateToTripCreationPreprocessedPins() {
        navigate(to: .tripCreation(.preprocessed))
    }

    public func navigateToTripCreationReview() {
        navigate(to: .tripCreation(.final))
    }
}

// MARK: Media Routing

extension AppRouter {
    public func navigateToMediaInfo(media: MediaItem) {
        navigate(to: .media(.info(media: media)))
    }

    public func navigateToLocalMediaInfo(media: LoadedMedia) {
        navigate(to: .media(.localInfo(media: media)))
    }
}

// MARK: - Profile Routing

extension AppRouter {
    public func navigateToEmailChange(email: String, action: EmailChangeAction) {
        navigate(to: .profile(.emailChange(email: email, action: action)))
    }

    public func navigateToStatistics() {
        navigate(to: .profile(.statistics))
    }

    public func navigateToTrips() {
        navigate(to: .profile(.trips))
    }

    public func navigateToPlacesWishlist() {
        navigate(to: .profile(.wishlist))
    }

    public func navigateToSavedMaps() {
        navigate(to: .profile(.saved))
    }

    public func navigateToNotifications() {
        navigate(to: .profile(.notifications))
    }

    public func navigateToAppearance() {
        navigate(to: .profile(.appearance))
    }
}

// MARK: - Wishlist Routing

extension AppRouter {
    public func navigateToWishlistElement(element: WishlistElement) {
        navigate(to: .wishlist(.element(element: element)))
    }

    public func navigateToWishlistElementCreation(action: WishlistCreationAction) {
        navigate(to: .wishlist(.creation(action: action)))
    }
}

// MARK: - Feed Routing

extension AppRouter {

}

// MARK: - PinInfo Routing
extension AppRouter {
    public func navigateToPinPlaceChange(pin: Pin, action: PlaceSaveAction) {
        navigate(to: .pinInfo(.placeChange(pin: pin, action: action)))
    }
}
