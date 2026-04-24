public struct ReviewPinDTO: Codable {
    public let pinId: String
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
        pinId: String,
        name: String? = nil,
        category: String? = nil,
        latitude: Double? = nil,
        longitude: Double? = nil,
        locationName: String? = nil,
        startTimeUnix: Int? = nil,
        endTimeUnix: Int? = nil,
        tags: [String]? = nil,
        issues: [String]? = nil,
        media: [ReviewPinMediaDTO]? = nil
    ) {
        self.pinId = pinId
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
        case pinId = "pin_id"
        case legacyPinId = "id"
        case name, category, latitude, longitude, tags, issues, media
        case locationName = "location_name"
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        if let pinIdFromPinId = try? container.decode(String.self, forKey: .pinId), !pinIdFromPinId.isEmpty {
            pinId = pinIdFromPinId
        } else if let pinIdFromLegacy = try? container.decode(String.self, forKey: .legacyPinId), !pinIdFromLegacy.isEmpty {
            pinId = pinIdFromLegacy
        } else {
            throw DecodingError.keyNotFound(
                CodingKeys.pinId,
                .init(codingPath: decoder.codingPath, debugDescription: "Review pin missing pin identifier (pin_id/id)")
            )
        }

        name = try container.decodeIfPresent(String.self, forKey: .name)
        category = try container.decodeIfPresent(String.self, forKey: .category)
        latitude = try container.decodeIfPresent(Double.self, forKey: .latitude)
        longitude = try container.decodeIfPresent(Double.self, forKey: .longitude)
        locationName = try container.decodeIfPresent(String.self, forKey: .locationName)
        startTimeUnix = try container.decodeIfPresent(Int.self, forKey: .startTimeUnix)
        endTimeUnix = try container.decodeIfPresent(Int.self, forKey: .endTimeUnix)
        tags = try container.decodeIfPresent([String].self, forKey: .tags)
        issues = try container.decodeIfPresent([String].self, forKey: .issues)
        media = try container.decodeIfPresent([ReviewPinMediaDTO].self, forKey: .media)
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(pinId, forKey: .pinId)
        try container.encodeIfPresent(name, forKey: .name)
        try container.encodeIfPresent(category, forKey: .category)
        try container.encodeIfPresent(latitude, forKey: .latitude)
        try container.encodeIfPresent(longitude, forKey: .longitude)
        try container.encodeIfPresent(locationName, forKey: .locationName)
        try container.encodeIfPresent(startTimeUnix, forKey: .startTimeUnix)
        try container.encodeIfPresent(endTimeUnix, forKey: .endTimeUnix)
        try container.encodeIfPresent(tags, forKey: .tags)
        try container.encodeIfPresent(issues, forKey: .issues)
        try container.encodeIfPresent(media, forKey: .media)
    }

}
