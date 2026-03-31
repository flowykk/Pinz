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
        true
//        accessToken != nil
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
        let data = Data(value.utf8)
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: key,
            kSecValueData: data
        ]
        SecItemDelete(query as CFDictionary)
        SecItemAdd(query as CFDictionary, nil)
    }

    private func read(key: String) -> String? {
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: key,
            kSecReturnData: true,
            kSecMatchLimit: kSecMatchLimitOne
        ]
        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func delete(key: String) {
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}
