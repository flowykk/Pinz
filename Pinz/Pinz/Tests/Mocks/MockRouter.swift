import Foundation
import PinzBase
import PinzDomain

final class MockRouter: AppRouting {
    var navigatedToMain = false
    var navigatedTripInfo: Trip?
    var navigatedProfile: User?
    var navigatedPinInfo: Pin?
    var navigatedToPinCreation = false
    var navigatedToTripMembers = false
    var navigatedToFeed = false
    var navigatedPinsList: Trip?
    var navigatedSelectablePinsList: Trip?
    var navigatedPostPreview: (trip: Trip, pins: [Pin])?
    var navigatedPostInfo: Post?
    var navigatedMediaInfo: MediaItem?
    var navigatedLocalMediaInfo: LoadedMedia?
    var navigatedEmailChange: (email: String, action: EmailChangeAction)?
    var navigatedToStatistics = false
    var navigatedToTrips = false
    var navigatedToPlacesWishlist = false
    var navigatedToSavedMaps = false
    var navigatedToNotifications = false
    var navigatedToAppearance = false
    var navigatedPinPlaceChange: (pin: Pin, action: PlaceSaveAction)?
    var navigatedWishlistElement: WishlistElement?
    var navigatedWishlistElementCreation: WishlistCreationAction?
    var popCallCount = 0
    var lastPopByCount = 0
    var tripInfoUpdateHandler: (() -> Void)?
    var currentProfileUpdateUser: User?
    var currentProfileUpdateCallCount: Int = 0
    private var currentProfileUpdateAction: ((User) -> Void)?

    func navigateToMain() { navigatedToMain = true }
    func navigateToTripInfo(trip: Trip, onTripUpdated: (() -> Void)?) {
        navigatedTripInfo = trip
        tripInfoUpdateHandler = onTripUpdated
    }
    func navigateToProfile(user: User) { navigatedProfile = user }
    func subscribeToCurrentProfileUpdates(_ action: @escaping (User) -> Void) {
        currentProfileUpdateAction = action
    }
    func notifyCurrentProfileUpdated(_ user: User) {
        currentProfileUpdateUser = user
        currentProfileUpdateCallCount += 1
        currentProfileUpdateAction?(user)
        currentProfileUpdateAction = nil
    }
    func clearCurrentProfileUpdates() {
        currentProfileUpdateAction = nil
        currentProfileUpdateUser = nil
    }
    func navigateToPinInfo(pin: Pin) { navigatedPinInfo = pin }
    func navigateToPinCreation() { navigatedToPinCreation = true }
    func navigateToTripMembers() { navigatedToTripMembers = true }
    func navigateToFeed() { navigatedToFeed = true }
    func navigateToPinsList(trip: Trip) { navigatedPinsList = trip }
    func navigateToSelectablePinsList(trip: Trip) { navigatedSelectablePinsList = trip }
    func navigateToPostPreview(trip: Trip, selectedPins: [Pin]) { navigatedPostPreview = (trip, selectedPins) }
    func navigateToPostInfo(post: Post) { navigatedPostInfo = post }
    func navigateToMediaInfo(media: MediaItem) { navigatedMediaInfo = media }
    func navigateToLocalMediaInfo(media: LoadedMedia) { navigatedLocalMediaInfo = media }
    func navigateToEmailChange(email: String, action: EmailChangeAction) { navigatedEmailChange = (email, action) }
    func navigateToStatistics() { navigatedToStatistics = true }
    func navigateToTrips() { navigatedToTrips = true }
    func navigateToPlacesWishlist() { navigatedToPlacesWishlist = true }
    func navigateToSavedMaps() { navigatedToSavedMaps = true }
    func navigateToNotifications() { navigatedToNotifications = true }
    func navigateToAppearance() { navigatedToAppearance = true }
    func navigateToPinPlaceChange(pin: Pin, action: PlaceSaveAction) { navigatedPinPlaceChange = (pin, action) }
    func navigateToWishlistElement(element: WishlistElement) { navigatedWishlistElement = element }
    func navigateToWishlistElementCreation(action: WishlistCreationAction) { navigatedWishlistElementCreation = action }

    func pop() {
        popCallCount += 1
        lastPopByCount = 1
    }

    func pop(by count: Int) {
        popCallCount += 1
        lastPopByCount = count
    }
}
