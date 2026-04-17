import Foundation

public struct ChangeEmailResponseDTO: Codable {
    public let success: Bool
    public let message: String?
    public let email: String?
    public let expiresAtUnix: Int?

    public init(success: Bool, message: String? = nil, email: String? = nil, expiresAtUnix: Int? = nil) {
        self.success = success
        self.message = message
        self.email = email
        self.expiresAtUnix = expiresAtUnix
    }

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case email
        case expiresAtUnix = "expires_at_unix"
    }
}
