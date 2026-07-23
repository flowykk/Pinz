public struct DesiredPlaceImageUploadResponseDTO: Codable {
    public let uploadUrl: String
    public let s3Key: String

    public init(uploadUrl: String, s3Key: String) {
        self.uploadUrl = uploadUrl
        self.s3Key = s3Key
    }

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case s3Key = "s3_key"
    }
}
