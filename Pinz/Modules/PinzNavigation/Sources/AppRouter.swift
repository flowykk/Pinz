import SwiftUI
import PinzDomain
import PinzBase

@MainActor @Observable
public final class AppRouter: AppRouting {
    public var path: [Route]
    
    @ObservationIgnored private var tripInfoUpdateHandler: (() -> Void)?
    @ObservationIgnored private var tripReloadHandler: (() -> Void)?
    @ObservationIgnored private var currentProfileUpdateHandler: ((User) -> Void)?

    public init(initialPath: [Route] = []) {
        self.path = initialPath
    }

    public func consumeTripInfoUpdateHandler() -> (() -> Void)? {
        defer { tripInfoUpdateHandler = nil }
        return tripInfoUpdateHandler
    }

    public func consumeTripReloadHandler() -> (() -> Void)? {
        defer { tripReloadHandler = nil }
        return tripReloadHandler
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

// MARK: - Profile update callbacks

extension AppRouter {
    public func subscribeToCurrentProfileUpdates(_ action: @escaping (User) -> Void) {
        currentProfileUpdateHandler = action
    }

    public func notifyCurrentProfileUpdated(_ user: User) {
        currentProfileUpdateHandler?(user)
        currentProfileUpdateHandler = nil
    }

    public func clearCurrentProfileUpdates() {
        currentProfileUpdateHandler = nil
    }
}

// MARK: - Trip Routing

extension AppRouter {
    public func navigateToTripInfo(trip: Trip, onTripUpdated: (() -> Void)?) {
        tripInfoUpdateHandler = onTripUpdated
        navigate(to: .trip(.info(trip: trip)))
    }

    public func navigateToProfile(user: User) {
        navigate(to: .trip(.profile(user: user)))
    }

    public func navigateToPinInfo(pin: Pin, updateAction: PinUpdateAction? = nil) {
        navigate(to: .trip(.pinInfo(pin: pin, updateAction: updateAction)))
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

    public func navigateToTripAddMedia(tripId: String) {
        navigate(to: .tripAddMedia(tripId: tripId))
    }
}

// MARK: - Trip reload callbacks

extension AppRouter {
    public func subscribeToTripReload(_ action: @escaping () -> Void) {
        tripReloadHandler = action
    }

    public func notifyTripReloadNeeded() {
        tripReloadHandler?()
        tripReloadHandler = nil
    }

    public func clearTripReloadUpdates() {
        tripReloadHandler = nil
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

    public func navigateToTripCreationPreprocessedPins(tripId: String, pins: RawPins) {
        navigate(to: .tripCreation(.preprocessed(tripId: tripId, pins: pins)))
    }

    public func navigateToTripCreationReview(tripId: String, pins: [Pin]) {
        navigate(to: .tripCreation(.final(tripId: tripId, pins: pins)))
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
    public func navigateToEmailChange(email: String, userId: String?, action: EmailChangeAction) {
        navigate(to: .profile(.emailChange(email: email, userId: userId, action: action)))
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

    public func navigateToStorageSettings() {
        navigate(to: .profile(.storageSettings))
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
