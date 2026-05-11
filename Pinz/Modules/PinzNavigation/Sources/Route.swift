import Foundation
import PinzDomain

public enum TripRoute: Hashable {
    case info(trip: Trip)
    case profile(user: User)
    case pinInfo(pin: Pin, updateAction: PinUpdateAction?, deleteAction: PinDeleteAction?)
    case pinCreation
    case members(tripId: String, participants: [TripParticipantDTO], currentUserId: String?)
    case publicProfile(userId: String)
    case publicWishlist(places: [DesiredPlace])
    case feed
}

public enum TripInfoRoute: Hashable {
    case pinsList(trip: Trip)
    case selectablePinsList(trip: Trip)
    case postPreview(trip: Trip, selectedPins: [Pin])
    case postInfo(post: Post)
    case savedTrip(trip: Trip)
}

public enum TripCreationRoute: Hashable {
    case initial
    case preprocessed(tripId: String, pins: RawPins)
    case final(tripId: String, pins: [Pin])
    case problems(tripId: String, pins: [Pin])
}

public enum ProfileRoute: Hashable {
    case emailChange(email: String, userId: String?, action: EmailChangeAction)

    case statistics

    case trips
    case wishlist
    case saved

    case storageSettings
    case notifications
    case appearance
}

public enum PinInfoRoute: Hashable {
    case placeChange(pin: Pin, action: PlaceSaveAction)
}

public enum MediaRoute: Hashable {
    case info(media: MediaItem, updateAction: MediaUpdateAction?)
    case localInfo(media: LoadedMedia)
}

public enum WishlistRoute: Hashable {
    case element(element: DesiredPlace)
    case creation(action: WishlistCreationAction)
}

public enum TripAddMediaRoute: Hashable {
    case start(tripId: String)
    case uploading(tripId: String, sessionId: String)
    case grouping(tripId: String, sessionId: String)
    case processing(tripId: String, sessionId: String)
    case review(tripId: String, sessionId: String)
}

public enum PinUploadRoute: Hashable {
    case start(tripId: String, targetPinId: String?)
    case processing(tripId: String, sessionId: String, targetPinId: String?)
    case review(tripId: String, sessionId: String, targetPinId: String?)
}

public enum Route: Hashable {
    case main
    case trip(TripRoute)
    case tripInfo(TripInfoRoute)
    case tripCreation(TripCreationRoute)
    case tripAddMedia(TripAddMediaRoute)
    case pinUpload(PinUploadRoute)
    case profile(ProfileRoute)
    case pinInfo(PinInfoRoute)
    case media(MediaRoute)
    case wishlist(WishlistRoute)
}
