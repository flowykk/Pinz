public struct PinUploadCommitResponseDTO: Codable {
    public let mediaId: String
    public let mediaCountInSession: Int

    public init(mediaId: String, mediaCountInSession: Int) {
        self.mediaId = mediaId
        self.mediaCountInSession = mediaCountInSession
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case mediaCountInSession = "media_count_in_session"
    }
}
