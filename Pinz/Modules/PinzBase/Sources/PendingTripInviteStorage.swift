import Foundation

/// Persists an invite token when the user opens an invite link before logging in.
public final class PendingTripInviteStorage {
    public static let shared = PendingTripInviteStorage()

    private let defaults = UserDefaults.standard
    private let key = "pinz.pendingTripInviteToken"

    private init() {}

    public func setPendingToken(_ token: String) {
        defaults.set(token, forKey: key)
    }

    public func peekPendingToken() -> String? {
        defaults.string(forKey: key)
    }

    /// Returns the stored token once and clears it.
    public func consumePendingToken() -> String? {
        guard let token = defaults.string(forKey: key), !token.isEmpty else {
            return nil
        }
        defaults.removeObject(forKey: key)
        return token
    }

    public func clear() {
        defaults.removeObject(forKey: key)
    }
}
