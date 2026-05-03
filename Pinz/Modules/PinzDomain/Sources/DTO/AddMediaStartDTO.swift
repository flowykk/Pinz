public struct AddMediaStartDTO: Codable {
    public let sessionId: String
    public let status: String
    public let joined: Bool
    public let uploadUrls: [UploadURLDTO]

    public init(sessionId: String, status: String, joined: Bool, uploadUrls: [UploadURLDTO]) {
        self.sessionId = sessionId
        self.status = status
        self.joined = joined
        self.uploadUrls = uploadUrls
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case status
        case joined
        case uploadUrls = "upload_urls"
    }
}
