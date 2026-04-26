public struct DesiredPlaceImageUploadResponseDTO: Codable {
    public let uploadUrl: String
    public let s3Key: String

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case s3Key = "s3_key"
    }
}
