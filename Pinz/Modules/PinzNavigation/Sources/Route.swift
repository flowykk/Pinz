import Foundation
import PinzDomain

public enum ProfileRoute: Hashable {
    case profile
    case emailChange(email: String, action: EmailChangeAction)
}

public enum Route: Hashable {
    case profile(ProfileRoute)
}
