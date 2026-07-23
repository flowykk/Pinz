public struct PinUpdateInputDTO {
    public let pinId: String
    public let name: String?
    public let description: String?
    public let category: String?
    public let privacyLevel: String?
    public let latitude: Double?
    public let longitude: Double?
    public let tags: [String]?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?

    public init(
        pinId: String,
        name: String? = nil,
        description: String? = nil,
        category: String? = nil,
        privacyLevel: String? = nil,
        latitude: Double? = nil,
        longitude: Double? = nil,
        tags: [String]? = nil,
        startTimeUnix: Int? = nil,
        endTimeUnix: Int? = nil
    ) {
        self.pinId = pinId
        self.name = name
        self.description = description
        self.category = category
        self.privacyLevel = privacyLevel
        self.latitude = latitude
        self.longitude = longitude
        self.tags = tags
        self.startTimeUnix = startTimeUnix
        self.endTimeUnix = endTimeUnix
    }
}
