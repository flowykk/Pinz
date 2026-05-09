import Foundation
import Security

public final class TokenStorage {
    public static let shared = TokenStorage()
    private init() {}

    private enum Key {
        static let accessToken = "pinz.accessToken"
        static let refreshToken = "pinz.refreshToken"
    }

    public var isAuthenticated: Bool {
        accessToken != nil
    }

    public var accessToken: String? {
        read(key: Key.accessToken)
    }

    public var refreshToken: String? {
        read(key: Key.refreshToken)
    }

    public func save(accessToken: String, refreshToken: String) {
        write(key: Key.accessToken, value: accessToken)
        write(key: Key.refreshToken, value: refreshToken)
        print("shared.refreshToken \(TokenStorage.shared.refreshToken)")
    }

    public func clear() {
        delete(key: Key.accessToken)
        delete(key: Key.refreshToken)
    }

    // MARK: - Storage

    private func write(key: String, value: String) {
        UserDefaults.standard.set(value, forKey: key)
    }

    private func read(key: String) -> String? {
        UserDefaults.standard.string(forKey: key)
    }

    private func delete(key: String) {
        UserDefaults.standard.removeObject(forKey: key)
    }
}
