import Foundation

public struct ConfirmAvatarUploadRequestDTO: Codable {
    public let s3Key: String

    public init(s3Key: String) {
        self.s3Key = s3Key
    }

    enum CodingKeys: String, CodingKey {
        case s3Key = "s3_key"
    }
}
