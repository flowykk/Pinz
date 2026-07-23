import Foundation

public final class PinUploadSessionStorage {
    public static let shared = PinUploadSessionStorage()

    private let defaults = UserDefaults.standard
    private let key = "pinz.pinUploadCreationSessions"

    private init() {}

    public func sessionId(forTripId tripId: String) -> String? {
        (defaults.dictionary(forKey: key) as? [String: String])?[tripId]
    }

    public func save(sessionId: String, forTripId tripId: String) {
        var dict = (defaults.dictionary(forKey: key) as? [String: String]) ?? [:]
        dict[tripId] = sessionId
        defaults.set(dict, forKey: key)
    }

    public func clear(forTripId tripId: String) {
        var dict = (defaults.dictionary(forKey: key) as? [String: String]) ?? [:]
        dict.removeValue(forKey: tripId)
        defaults.set(dict, forKey: key)
    }

    /// Clears all stored creation uploads (e.g. UI tests / simulator resets).
    public func clearAll() {
        defaults.removeObject(forKey: key)
    }
}
