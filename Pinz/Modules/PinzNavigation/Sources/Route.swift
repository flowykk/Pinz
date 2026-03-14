import Foundation
import PinzDomain

public enum TripRoute: Hashable {
    case info(trip: Trip)
    case profile(user: User)
    case pinInfo(pin: Pin)
    case pinCreation
    case members
    case feed
}

public enum TripInfoRoute: Hashable {
    case pinsList(trip: Trip)
    case selectablePinsList(trip: Trip)
    case postPreview(trip: Trip, selectedPins: [Pin])
}

public enum ProfileRoute: Hashable {
    case emailChange(email: String, action: EmailChangeAction)

    case statistics

    case trips
    case wishlist
    case saved

    case notifications
    case appearance
}

public enum PinInfoRoute: Hashable {
    case placeChange(pin: Pin, action: PlaceSaveAction)
}

public enum MediaRoute: Hashable {
    case info(media: MediaItem)
    case localInfo(media: LoadedMedia)
}

public enum Route: Hashable {
    case main
    case trip(TripRoute)
    case tripInfo(TripInfoRoute)
    case profile(ProfileRoute)
    case pinInfo(PinInfoRoute)
    case media(MediaRoute)
}
