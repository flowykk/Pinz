import Foundation

public extension Notification.Name {

    static let pinzDidAuthenticate = Notification.Name("io.tuist.hse.Pinz.pinzDidAuthenticate")

    /// Posted after tokens are cleared when the session is no longer valid (logout, 401, etc.).
    static let pinzSessionInvalidated = Notification.Name("io.tuist.hse.Pinz.pinzSessionInvalidated")
}
