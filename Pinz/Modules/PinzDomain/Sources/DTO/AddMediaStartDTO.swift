import Foundation

public struct AddMediaStartDTO: Codable {
    public let sessionId: String
    public let status: String
    public let uploadUrls: [UploadURLDTO]

    public init(sessionId: String, status: String, uploadUrls: [UploadURLDTO]) {
        self.sessionId = sessionId
        self.status = status
        self.uploadUrls = uploadUrls
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case status
        case uploadUrls = "upload_urls"
    }
}
