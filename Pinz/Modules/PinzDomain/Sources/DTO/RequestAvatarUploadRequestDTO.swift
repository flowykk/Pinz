import Foundation

public struct RequestAvatarUploadRequestDTO: Codable {
    public let filename: String
    public let contentType: String

    public init(filename: String, contentType: String) {
        self.filename = filename
        self.contentType = contentType
    }

    enum CodingKeys: String, CodingKey {
        case filename
        case contentType = "content_type"
    }
}
