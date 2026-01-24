import Foundation
import PinzDomain

public enum TripRoute: Hashable {
    case info(trip: Trip)
    case profile(user: User)
    case pinInfo(pin: Pin)
}

public enum ProfileRoute: Hashable {
    case emailChange(email: String, action: EmailChangeAction)
    case appearance
}

public enum Route: Hashable {
    case trip(TripRoute)
    case profile(ProfileRoute)
}
