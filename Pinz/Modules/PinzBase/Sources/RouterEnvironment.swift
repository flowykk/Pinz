import SwiftUI
import PinzDomain

public protocol AppRouting {
    func navigateToTripInfo(trip: Trip)
    func navigateToProfile(user: User)
    func navigateToPinInfo(pin: Pin)
    func navigateToPinCreation()
    func navigateToTripMembers()
    func navigateToFeed()

    func navigateToPinsList(trip: Trip)
    func navigateToSelectablePinsList(trip: Trip)
    func navigateToPostPreview(trip: Trip, selectedPins: [Pin])

    func navigateToMediaInfo(media: MediaItem)

    func navigateToEmailChange(email: String, action: EmailChangeAction)
    func navigateToStatistics()
    func navigateToTrips()
    func navigateToPlacesWishlist()
    func navigateToSavedMaps()
    func navigateToNotifications()
    func navigateToAppearance()

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
