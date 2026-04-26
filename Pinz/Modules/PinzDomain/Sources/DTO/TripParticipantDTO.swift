public struct TripParticipantDTO: Codable {
    public let userId: String
    public let username: String
    public let avatarUrl: String?
    public let role: String?
    public let privacyLevel: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case username
        case avatarUrl = "avatar_url"
        case role
        case privacyLevel = "privacy_level"
    }
}
