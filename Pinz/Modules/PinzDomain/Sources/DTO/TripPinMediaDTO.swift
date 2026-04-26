public struct TripPinMediaDTO: Codable {
    public let mediaId: String
    public let url: String
    public let mediaType: String?
    public let privacyLevel: String?
    public let capturedAtUnix: Int?

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case url
        case mediaType = "media_type"
        case privacyLevel = "privacy_level"
        case capturedAtUnix = "captured_at_unix"
    }
}
