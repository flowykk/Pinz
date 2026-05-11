public struct PinUploadFinalizeInputDTO {
    public let name: String?
    public let description: String?
    public let category: String?
    public let latitude: Double?
    public let longitude: Double?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?
    public let tags: [String]?
    public let tagsSet: Bool?
    public let mediaToDelete: [String]

    public init(
        name: String? = nil,
        description: String? = nil,
        category: String? = nil,
        latitude: Double? = nil,
        longitude: Double? = nil,
        startTimeUnix: Int? = nil,
        endTimeUnix: Int? = nil,
        tags: [String]? = nil,
        tagsSet: Bool? = nil,
        mediaToDelete: [String] = []
    ) {
        self.name = name
        self.description = description
        self.category = category
        self.latitude = latitude
        self.longitude = longitude
        self.startTimeUnix = startTimeUnix
        self.endTimeUnix = endTimeUnix
        self.tags = tags
        self.tagsSet = tagsSet
        self.mediaToDelete = mediaToDelete
    }
}
