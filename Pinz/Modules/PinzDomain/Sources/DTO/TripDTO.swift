public struct TripDTO: Codable {
    public let id: String
    public let name: String
    public let description: String?
    public let category: String?
    public let season: String?
    public let coverUrl: String?
    public let ownerUserId: String
    public let privacyLevel: String?
    public let status: String?
    public let isPublished: Bool
    public let isGenerated: Bool
    public let likesCount: Int
    public let dislikesCount: Int
    public let startDateUnix: Int?
    public let endDateUnix: Int?
    public let createdAtUnix: Int
    public let updatedAtUnix: Int

    public init(
        id: String, name: String, description: String?, category: String?,
        season: String?, coverUrl: String?, ownerUserId: String, privacyLevel: String?,
        status: String?, isPublished: Bool, isGenerated: Bool, likesCount: Int,
        dislikesCount: Int, startDateUnix: Int?, endDateUnix: Int?,
        createdAtUnix: Int, updatedAtUnix: Int
    ) {
        self.id = id; self.name = name; self.description = description
        self.category = category; self.season = season; self.coverUrl = coverUrl
        self.ownerUserId = ownerUserId; self.privacyLevel = privacyLevel
        self.status = status; self.isPublished = isPublished
        self.isGenerated = isGenerated; self.likesCount = likesCount
        self.dislikesCount = dislikesCount; self.startDateUnix = startDateUnix
        self.endDateUnix = endDateUnix; self.createdAtUnix = createdAtUnix
        self.updatedAtUnix = updatedAtUnix
    }

    enum CodingKeys: String, CodingKey {
        case id, name, description, category, season, status
        case coverUrl = "cover_url"
        case ownerUserId = "owner_user_id"
        case privacyLevel = "privacy_level"
        case isPublished = "is_published"
        case isGenerated = "is_generated"
        case likesCount = "likes_count"
        case dislikesCount = "dislikes_count"
        case startDateUnix = "start_date_unix"
        case endDateUnix = "end_date_unix"
        case createdAtUnix = "created_at_unix"
        case updatedAtUnix = "updated_at_unix"
    }
}
