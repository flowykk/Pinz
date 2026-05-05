public struct DesiredPlaceDTO: Codable {
    public let id: String
    public let name: String
    public let description: String
    public let imageUrl: String?
    public let createdAt: Int

    public init(id: String, name: String, description: String, imageUrl: String?, createdAt: Int) {
        self.id = id
        self.name = name
        self.description = description
        self.imageUrl = imageUrl
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case id, name, description
        case imageUrl = "image_url"
        case createdAt = "created_at"
    }

    public func toDesiredPlace() -> DesiredPlace {
        DesiredPlace(id: id, name: name, description: description, imageUrl: imageUrl, createdAt: createdAt)
    }
}
