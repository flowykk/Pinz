public struct DraftPinMediaDTO: Codable {
    public let mediaId: String
    public let type: String
    public let url: String

    public init(mediaId: String, type: String, url: String) {
        self.mediaId = mediaId
        self.type = type
        self.url = url
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case type
        case url
    }
}
