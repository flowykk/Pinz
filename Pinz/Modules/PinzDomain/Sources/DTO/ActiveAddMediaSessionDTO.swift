public struct ActiveAddMediaSessionDTO: Codable {
    public let sessionId: String
    public let currentInitiator: PublicUserProfileDTO?
    public let initiatorAssignedAt: String?
    public let mediaCountInSession: Int?
    public let takeoverAvailableAt: String?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case currentInitiator = "current_initiator"
        case initiatorAssignedAt = "initiator_assigned_at"
        case mediaCountInSession = "media_count_in_session"
        case takeoverAvailableAt = "takeover_available_at"
    }
}

public struct PublicUserProfileDTO: Codable {
    public let userId: String
    public let username: String
    public let avatarUrl: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case username
        case avatarUrl = "avatar_url"
    }
}
