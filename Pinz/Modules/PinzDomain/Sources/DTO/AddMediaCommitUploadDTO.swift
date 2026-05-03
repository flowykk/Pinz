public struct AddMediaCommitUploadDTO: Codable {
    public let mediaId: String
    public let mediaCountInSession: Int
    public let remainingSlots: Int

    public init(mediaId: String, mediaCountInSession: Int, remainingSlots: Int) {
        self.mediaId = mediaId
        self.mediaCountInSession = mediaCountInSession
        self.remainingSlots = remainingSlots
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case mediaCountInSession = "media_count_in_session"
        case remainingSlots = "remaining_slots"
    }
}
