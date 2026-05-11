public struct PinUploadReviewResponseDTO: Codable {
    public let sessionId: String
    public let processingStatus: String
    public let draft: PinUploadDraftDTO?
    public let similar: [[String]]?

    public init(
        sessionId: String,
        processingStatus: String,
        draft: PinUploadDraftDTO?,
        similar: [[String]]?
    ) {
        self.sessionId = sessionId
        self.processingStatus = processingStatus
        self.draft = draft
        self.similar = similar
    }

    enum CodingKeys: String, CodingKey {
        case draft, similar
        case sessionId = "session_id"
        case processingStatus = "processing_status"
    }
}
