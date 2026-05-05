public struct PublicUserProfileResponseDTO: Codable {
    public let id: String
    public let username: String
    public let avatarUrl: String?
    public let createdAt: Int
    public let desiredPlaces: [DesiredPlaceDTO]

    public init(id: String, username: String, avatarUrl: String?, createdAt: Int, desiredPlaces: [DesiredPlaceDTO]) {
        self.id = id
        self.username = username
        self.avatarUrl = avatarUrl
        self.createdAt = createdAt
        self.desiredPlaces = desiredPlaces
    }

    enum CodingKeys: String, CodingKey {
        case id, username
        case avatarUrl = "avatar_url"
        case createdAt = "created_at"
        case desiredPlaces = "desired_places"
    }
}
