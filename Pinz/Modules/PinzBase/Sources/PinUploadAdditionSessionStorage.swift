import Foundation

public final class PinUploadAdditionSessionStorage {
    public static let shared = PinUploadAdditionSessionStorage()

    private let defaults = UserDefaults.standard
    private let key = "pinz.pinUploadAdditionSessions"

    private init() {}

    private func storageKey(tripId: String, pinId: String) -> String {
        "\(tripId):\(pinId)"
    }

    public func sessionId(tripId: String, pinId: String) -> String? {
        let dict = defaults.dictionary(forKey: key) as? [String: String]
        return dict?[storageKey(tripId: tripId, pinId: pinId)]
    }

    public func save(sessionId: String, tripId: String, pinId: String) {
        var dict = (defaults.dictionary(forKey: key) as? [String: String]) ?? [:]
        dict[storageKey(tripId: tripId, pinId: pinId)] = sessionId
        defaults.set(dict, forKey: key)
    }

    public func clear(tripId: String, pinId: String) {
        var dict = (defaults.dictionary(forKey: key) as? [String: String]) ?? [:]
        dict.removeValue(forKey: storageKey(tripId: tripId, pinId: pinId))
        defaults.set(dict, forKey: key)
    }
}
