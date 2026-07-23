public struct PinUploadStartResponseDTO: Codable {
    public let sessionId: String
    public let uploadUrls: [UploadURLDTO]

    public init(sessionId: String, uploadUrls: [UploadURLDTO]) {
        self.sessionId = sessionId
        self.uploadUrls = uploadUrls
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case uploadUrls = "upload_urls"
    }
}
