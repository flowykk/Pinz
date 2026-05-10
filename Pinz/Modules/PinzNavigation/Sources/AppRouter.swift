import SwiftUI
import PinzDomain
import PinzBase

@MainActor @Observable
public final class AppRouter: AppRouting {
    public var path: [Route]
    
    @ObservationIgnored private var tripInfoUpdateHandler: (() -> Void)?
    @ObservationIgnored private var currentProfileUpdateHandler: ((User) -> Void)?
    @ObservationIgnored private var tripCreationDraftPins: [String: [Pin]] = [:]

    public init(initialPath: [Route] = []) {
        self.path = initialPath
    }

    public func consumeTripInfoUpdateHandler() -> (() -> Void)? {
        defer { tripInfoUpdateHandler = nil }
        return tripInfoUpdateHandler
    }

    public func navigateToMain() {
        path = [.main]
    }

    public func navigateToAuthenticationRoot() {
        path = []
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

    public func popToRoot() {
        path = [.main]
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

    public func navigateToPinInfo(pin: Pin, updateAction: PinUpdateAction? = nil, deleteAction: PinDeleteAction? = nil) {
        navigate(to: .trip(.pinInfo(pin: pin, updateAction: updateAction, deleteAction: deleteAction)))
    }

    public func navigateToPinCreation() {
        navigate(to: .trip(.pinCreation))
    }

    public func navigateToTripMembers(tripId: String, participants: [TripParticipantDTO], currentUserId: String?) {
        navigate(to: .trip(.members(tripId: tripId, participants: participants, currentUserId: currentUserId)))
    }

    public func navigateToPublicProfile(userId: String) {
        navigate(to: .trip(.publicProfile(userId: userId)))
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

    public func navigateToPostInfo(post: Post) {
        navigate(to: .tripInfo(.postInfo(post: post)))
    }

    public func navigateToSavedTripDetail(trip: Trip) {
        navigate(to: .tripInfo(.savedTrip(trip: trip)))
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

    public func navigateToTripCreationProblems(tripId: String, pins: [Pin]) {
        navigate(to: .tripCreation(.problems(tripId: tripId, pins: pins)))
    }

    public func setTripCreationDraftPins(_ pins: [Pin], for tripId: String) {
        tripCreationDraftPins[tripId] = pins
    }

    public func tripCreationDraftPins(for tripId: String) -> [Pin]? {
        tripCreationDraftPins[tripId]
    }

    public func clearTripCreationDraftPins(for tripId: String) {
        tripCreationDraftPins[tripId] = nil
    }
}

// MARK: Media Routing

extension AppRouter {
    public func navigateToMediaInfo(media: MediaItem, updateAction: MediaUpdateAction? = nil) {
        navigate(to: .media(.info(media: media, updateAction: updateAction)))
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
    public func navigateToWishlistElement(element: DesiredPlace) {
        navigate(to: .wishlist(.element(element: element)))
    }

    public func navigateToWishlistElementCreation(action: WishlistCreationAction) {
        navigate(to: .wishlist(.creation(action: action)))
    }

    public func navigateToPublicWishlist(places: [DesiredPlace]) {
        navigate(to: .trip(.publicWishlist(places: places)))
    }
}

// MARK: - AddMedia Routing

extension AppRouter {
    public func navigateToAddMediaStart(tripId: String) {
        navigate(to: .tripAddMedia(.start(tripId: tripId)))
    }

    public func navigateToAddMediaUploading(tripId: String, sessionId: String) {
        navigate(to: .tripAddMedia(.uploading(tripId: tripId, sessionId: sessionId)))
    }

    public func navigateToAddMediaGrouping(tripId: String, sessionId: String) {
        navigate(to: .tripAddMedia(.grouping(tripId: tripId, sessionId: sessionId)))
    }

    public func navigateToAddMediaProcessing(tripId: String, sessionId: String) {
        navigate(to: .tripAddMedia(.processing(tripId: tripId, sessionId: sessionId)))
    }

    public func navigateToAddMediaReview(tripId: String, sessionId: String) {
        navigate(to: .tripAddMedia(.review(tripId: tripId, sessionId: sessionId)))
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
