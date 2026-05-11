import Foundation

/// Persists active pin-upload creation `session_id` per trip.
///
/// Контракт:
/// - Сохранение сразу после успеха `pinUploadStart` (до S3 PUT).
///   Если приложение убьют между start и commit — сессия живая на бэке,
///   восстановим через `/review`.
/// - Удаление только после успеха `finalize` или `cancel`
///   (на 409 closed тоже удаляем — сессия уже мёртвая).
/// - Хранится только creation-сессия (`target_pin_id == null`).
///   Addition-флоу — в отдельной структуре, когда дойдём.
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
}
