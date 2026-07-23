public struct TripPinMediaDTO: Codable {
    public let mediaId: String
    public let url: String
    public let mediaType: String?
    public let privacyLevel: String?
    public let capturedAtUnix: Int?

    public init(mediaId: String, url: String, mediaType: String?, privacyLevel: String?, capturedAtUnix: Int?) {
        self.mediaId = mediaId
        self.url = url
        self.mediaType = mediaType
        self.privacyLevel = privacyLevel
        self.capturedAtUnix = capturedAtUnix
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case url
        case mediaType = "media_type"
        case privacyLevel = "privacy_level"
        case capturedAtUnix = "captured_at_unix"
    }
}
