import Foundation

public struct ChangeEmailRequestDTO: Codable {
    public let userId: String?
    public let newEmail: String

    public init(userId: String? = nil, newEmail: String) {
        self.userId = userId
        self.newEmail = newEmail
    }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case newEmail = "new_email"
    }
}
