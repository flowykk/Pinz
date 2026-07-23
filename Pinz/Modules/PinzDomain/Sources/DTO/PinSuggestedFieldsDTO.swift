public struct PinSuggestedFieldsDTO: Codable {
    public let name: String?
    public let category: String?
    public let tags: [String]?
    public let latitude: Double?
    public let longitude: Double?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?

    public init(
        name: String?,
        category: String?,
        tags: [String]?,
        latitude: Double?,
        longitude: Double?,
        startTimeUnix: Int?,
        endTimeUnix: Int?
    ) {
        self.name = name
        self.category = category
        self.tags = tags
        self.latitude = latitude
        self.longitude = longitude
        self.startTimeUnix = startTimeUnix
        self.endTimeUnix = endTimeUnix
    }

    enum CodingKeys: String, CodingKey {
        case name, category, tags, latitude, longitude
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
    }
}
