import SwiftUI
import PinzDomain

public protocol AppRouting: AnyObject {
    func navigateToMain()
    func navigateToAuthenticationRoot()

    func navigateToTripInfo(trip: Trip, onTripUpdated: (() -> Void)?)
    func navigateToProfile(user: User)
    func navigateToPinInfo(pin: Pin, updateAction: PinUpdateAction?, deleteAction: PinDeleteAction?)
    func navigateToPinCreation()
    func navigateToTripMembers(tripId: String, participants: [TripParticipantDTO], currentUserId: String?)
    func navigateToPublicProfile(userId: String)
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
    func navigateToSavedTripDetail(trip: Trip)

    func navigateToMediaInfo(
        media: MediaItem,
        updateAction: MediaUpdateAction?,
        pinIdForServerMediaDelete: String?,
        pinResponseAction: PinResponseAction?,
        allowsMediaPrivacyChange: Bool
    )
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

    func subscribeToTripPinsReload(_ action: @escaping (String) -> Void)
    func notifyTripPinsReload(tripId: String)
    func clearTripPinsReloadSubscription()

    func navigateToPinPlaceChange(pin: Pin, action: PlaceSaveAction)

    func navigateToWishlistElement(element: DesiredPlace)
    func navigateToWishlistElementCreation(action: WishlistCreationAction)
    func navigateToPublicWishlist(places: [DesiredPlace])

    func navigateToAddMediaStart(tripId: String)
    func navigateToAddMediaUploading(tripId: String, sessionId: String)
    func navigateToAddMediaGrouping(tripId: String, sessionId: String)
    func navigateToAddMediaProcessing(tripId: String, sessionId: String)
    func navigateToAddMediaReview(tripId: String, sessionId: String)

    func navigateToAddMediaProblems(tripId: String, sessionId: String)
    func setAddMediaReviewDraftPins(_ pins: [Pin], forSessionId sessionId: String)
    func addMediaReviewDraftPins(forSessionId sessionId: String) -> [Pin]?
    func clearAddMediaReviewDraftPins(forSessionId sessionId: String)

    func navigateToPinUploadStart(tripId: String, targetPinId: String?)
    func navigateToPinUploadProcessing(tripId: String, sessionId: String, targetPinId: String?)
    func navigateToPinUploadReview(tripId: String, sessionId: String, targetPinId: String?)
    func navigateToPinUploadProblems(tripId: String, sessionId: String, targetPinId: String?)
    func setPinUploadReviewDraftPin(_ pin: Pin, forSessionId sessionId: String)
    func pinUploadReviewDraftPin(forSessionId sessionId: String) -> Pin?
    func clearPinUploadReviewDraftPin(forSessionId sessionId: String)

    func pop()
    func pop(by count: Int)
    func popToRoot()

    func popAllPinUploadRoutes()

    func setPinUploadAdditionSuccessHandler(_ handler: ((Pin) -> Void)?)
    func notifyPinUploadAdditionSuccess(_ pin: Pin)
}

public extension AppRouting {
    func navigateToMediaInfo(media: MediaItem, updateAction: MediaUpdateAction? = nil) {
        navigateToMediaInfo(
            media: media,
            updateAction: updateAction,
            pinIdForServerMediaDelete: nil,
            pinResponseAction: nil,
            allowsMediaPrivacyChange: true
        )
    }

    func navigateToMediaInfo(
        media: MediaItem,
        updateAction: MediaUpdateAction?,
        pinIdForServerMediaDelete: String?,
        pinResponseAction: PinResponseAction?
    ) {
        navigateToMediaInfo(
            media: media,
            updateAction: updateAction,
            pinIdForServerMediaDelete: pinIdForServerMediaDelete,
            pinResponseAction: pinResponseAction,
            allowsMediaPrivacyChange: true
        )
    }

    func navigateToPinUploadStart(tripId: String) {
        navigateToPinUploadStart(tripId: tripId, targetPinId: nil)
    }

    func navigateToPinUploadProcessing(tripId: String, sessionId: String) {
        navigateToPinUploadProcessing(tripId: tripId, sessionId: sessionId, targetPinId: nil)
    }

    func navigateToPinUploadReview(tripId: String, sessionId: String) {
        navigateToPinUploadReview(tripId: tripId, sessionId: sessionId, targetPinId: nil)
    }
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
