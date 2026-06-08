import CryptoKit
import Foundation

public enum ContentHasher {
    public static func sha256Hex(of data: Data) -> String {
        SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
    }

    public static func sha256Hex(ofFile url: URL) -> String? {
        guard let data = try? Data(contentsOf: url) else { return nil }
        return sha256Hex(of: data)
    }

    public static func sha256Hex(from prepared: PreparedUpload) -> String? {
        switch prepared.body {
        case let .data(data):
            return sha256Hex(of: data)
        case let .file(url):
            return sha256Hex(ofFile: url)
        }
    }
}
