public struct ReviewPinDTO: Codable {
    public let id: String
    public let name: String?
    public let category: String?
    public let latitude: Double?
    public let longitude: Double?
    public let locationName: String?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?
    public let tags: [String]?
    public let issues: [String]?
    public let media: [ReviewPinMediaDTO]?

    public init(
        id: String,
        name: String?,
        category: String?,
        latitude: Double?,
        longitude: Double?,
        locationName: String?,
        startTimeUnix: Int?,
        endTimeUnix: Int?,
        tags: [String]?,
        issues: [String]?,
        media: [ReviewPinMediaDTO]?
    ) {
        self.id = id
        self.name = name
        self.category = category
        self.latitude = latitude
        self.longitude = longitude
        self.locationName = locationName
        self.startTimeUnix = startTimeUnix
        self.endTimeUnix = endTimeUnix
        self.tags = tags
        self.issues = issues
        self.media = media
    }

    enum CodingKeys: String, CodingKey {
        case id
        case name, category, latitude, longitude, tags, issues, media
        case locationName = "location_name"
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
    }
}
