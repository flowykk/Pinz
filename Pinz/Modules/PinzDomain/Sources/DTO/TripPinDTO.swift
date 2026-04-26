public struct TripPinDTO: Codable {
    public let id: String
    public let tripId: String?
    public let name: String?
    public let description: String?
    public let category: String?
    public let latitude: Double?
    public let longitude: Double?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?
    public let tags: [String]?
    public let privacyLevel: String?
    public let media: [TripPinMediaDTO]?

    enum CodingKeys: String, CodingKey {
        case id, name, description, category, latitude, longitude, tags, media
        case tripId = "trip_id"
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
        case privacyLevel = "privacy_level"
    }
}
