public struct ReviewPinMediaDTO: Codable, Hashable {
    public let mediaId: String
    public let url: String
    public let privacyLevel: String?

    public init(mediaId: String, url: String, privacyLevel: String?) {
        self.mediaId = mediaId
        self.url = url
        self.privacyLevel = privacyLevel
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case url
        case privacyLevel = "privacy_level"
    }
}
