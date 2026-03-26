public struct UploadURLDTO: Codable {
    public let clientId: String
    public let s3Key: String
    public let url: String

    public init(clientId: String, s3Key: String, url: String) {
        self.clientId = clientId
        self.s3Key = s3Key
        self.url = url
    }

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case s3Key = "s3_key"
        case url
    }
}
