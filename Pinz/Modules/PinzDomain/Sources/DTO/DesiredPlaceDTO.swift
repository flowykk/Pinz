public struct DesiredPlaceDTO: Codable {
    public let id: String
    public let name: String
    public let description: String
    public let imageUrl: String?
    public let createdAt: Int

    enum CodingKeys: String, CodingKey {
        case id, name, description
        case imageUrl = "image_url"
        case createdAt = "created_at"
    }

    public func toDesiredPlace() -> DesiredPlace {
        DesiredPlace(id: id, name: name, description: description, imageUrl: imageUrl, createdAt: createdAt)
    }
}
