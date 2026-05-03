public struct AddMediaSessionMediaDTO: Codable {
    public let sessionId: String
    public let media: [AddMediaSessionMediaEntryDTO]
    public let mediaCountInSession: Int

    public init(sessionId: String, media: [AddMediaSessionMediaEntryDTO], mediaCountInSession: Int) {
        self.sessionId = sessionId
        self.media = media
        self.mediaCountInSession = mediaCountInSession
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case media
        case mediaCountInSession = "media_count_in_session"
    }
}

public struct AddMediaSessionMediaEntryDTO: Codable, Hashable {
    public let mediaId: String
    public let url: String
    public let type: String
    public let actorUserId: String
    public let uploadedAt: String

    public init(mediaId: String, url: String, type: String, actorUserId: String, uploadedAt: String) {
        self.mediaId = mediaId
        self.url = url
        self.type = type
        self.actorUserId = actorUserId
        self.uploadedAt = uploadedAt
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case url
        case type
        case actorUserId = "actor_user_id"
        case uploadedAt = "uploaded_at"
    }
}
