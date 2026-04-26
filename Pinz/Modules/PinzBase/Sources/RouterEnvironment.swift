import SwiftUI
import PinzDomain

public protocol AppRouting {
    func navigateToMain()

    func navigateToTripInfo(trip: Trip, onTripUpdated: (() -> Void)?)
    func navigateToProfile(user: User)
    func navigateToPinInfo(pin: Pin, updateAction: PinUpdateAction?)
    func navigateToPinCreation()
    func navigateToTripMembers()
    func navigateToFeed()

    func navigateToTripCreationInitial()
    func navigateToTripCreationPreprocessedPins(tripId: String, pins: RawPins)
    func navigateToTripCreationReview(tripId: String, pins: [Pin])
    func navigateToTripCreationProblems(tripId: String, pins: [Pin])
    func setTripCreationDraftPins(_ pins: [Pin], for tripId: String)
    func tripCreationDraftPins(for tripId: String) -> [Pin]?
    func clearTripCreationDraftPins(for tripId: String)

    func navigateToPinsList(trip: Trip)
    func navigateToSelectablePinsList(trip: Trip)
    func navigateToPostPreview(trip: Trip, selectedPins: [Pin])
    func navigateToPostInfo(post: Post)

    func navigateToMediaInfo(media: MediaItem, updateAction: MediaUpdateAction?)
    func navigateToLocalMediaInfo(media: LoadedMedia)

    func navigateToEmailChange(email: String, userId: String?, action: EmailChangeAction)
    func navigateToStatistics()
    func navigateToTrips()
    func navigateToPlacesWishlist()
    func navigateToSavedMaps()
    func navigateToStorageSettings()
    func navigateToNotifications()
    func navigateToAppearance()

    func subscribeToCurrentProfileUpdates(_ action: @escaping (User) -> Void)
    func notifyCurrentProfileUpdated(_ user: User)
    func clearCurrentProfileUpdates()

    func navigateToPinPlaceChange(pin: Pin, action: PlaceSaveAction)

    func navigateToWishlistElement(element: WishlistElement)
    func navigateToWishlistElementCreation(action: WishlistCreationAction)

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
