import Foundation

public struct ProfileResponseDTO: Codable {
    public let id: String?
    public let userId: String?
    public let username: String?
    public let nickname: String?
    public let email: String?
    public let avatarUrl: String?

    public init(
        id: String? = nil,
        userId: String? = nil,
        username: String? = nil,
        nickname: String? = nil,
        email: String? = nil,
        avatarUrl: String? = nil
    ) {
        self.id = id
        self.userId = userId
        self.username = username
        self.nickname = nickname
        self.email = email
        self.avatarUrl = avatarUrl
    }

    enum CodingKeys: String, CodingKey {
        case id
        case userId = "user_id"
        case username
        case nickname
        case email
        case avatarUrl = "avatar_url"
    }
}
