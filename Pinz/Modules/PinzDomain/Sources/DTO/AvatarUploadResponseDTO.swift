import Foundation

public struct AvatarUploadResponseDTO: Codable {
    public let uploadUrl: String?
    public let s3Key: String?
    public let expiresAtUnix: Int?

    public init(uploadUrl: String? = nil, s3Key: String? = nil, expiresAtUnix: Int? = nil) {
        self.uploadUrl = uploadUrl
        self.s3Key = s3Key
        self.expiresAtUnix = expiresAtUnix
    }

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case s3Key = "s3_key"
        case expiresAtUnix = "expires_at_unix"
    }
}
