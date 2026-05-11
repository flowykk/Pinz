public struct PinUploadProcessResponseDTO: Codable {
    public let sessionId: String
    public let processingStatus: String

    public init(sessionId: String, processingStatus: String) {
        self.sessionId = sessionId
        self.processingStatus = processingStatus
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case processingStatus = "processing_status"
    }
}
