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
    public let participantsCount: Int?
    public let mediaCount: Int?
    public let startDateUnix: Int?
    public let endDateUnix: Int?
    public let createdAtUnix: Int
    public let updatedAtUnix: Int
    public let isNameCensored: Bool
    public let isDescriptionCensored: Bool

    public init(
        id: String, name: String, description: String?, category: String?,
        season: String?, coverUrl: String?, ownerUserId: String, privacyLevel: String?,
        status: String?, isPublished: Bool, isGenerated: Bool, likesCount: Int,
        dislikesCount: Int, participantsCount: Int? = nil, mediaCount: Int? = nil,
        startDateUnix: Int?, endDateUnix: Int?,
        createdAtUnix: Int, updatedAtUnix: Int,
        isNameCensored: Bool = false, isDescriptionCensored: Bool = false
    ) {
        self.id = id; self.name = name; self.description = description
        self.category = category; self.season = season; self.coverUrl = coverUrl
        self.ownerUserId = ownerUserId; self.privacyLevel = privacyLevel
        self.status = status; self.isPublished = isPublished
        self.isGenerated = isGenerated; self.likesCount = likesCount
        self.dislikesCount = dislikesCount
        self.participantsCount = participantsCount
        self.mediaCount = mediaCount
        self.startDateUnix = startDateUnix
        self.endDateUnix = endDateUnix; self.createdAtUnix = createdAtUnix
        self.updatedAtUnix = updatedAtUnix
        self.isNameCensored = isNameCensored
        self.isDescriptionCensored = isDescriptionCensored
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
        case participantsCount = "participants_count"
        case mediaCount = "media_count"
        case startDateUnix = "start_date_unix"
        case endDateUnix = "end_date_unix"
        case createdAtUnix = "created_at_unix"
        case updatedAtUnix = "updated_at_unix"
        case isNameCensored = "name_censored"
        case isDescriptionCensored = "description_censored"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decode(String.self, forKey: .name)
        description = try c.decodeIfPresent(String.self, forKey: .description)
        category = try c.decodeIfPresent(String.self, forKey: .category)
        season = try c.decodeIfPresent(String.self, forKey: .season)
        coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl)
        ownerUserId = try c.decode(String.self, forKey: .ownerUserId)
        privacyLevel = try c.decodeIfPresent(String.self, forKey: .privacyLevel)
        status = try c.decodeIfPresent(String.self, forKey: .status)
        isPublished = try c.decode(Bool.self, forKey: .isPublished)
        isGenerated = try c.decode(Bool.self, forKey: .isGenerated)
        likesCount = try c.decode(Int.self, forKey: .likesCount)
        dislikesCount = try c.decode(Int.self, forKey: .dislikesCount)
        participantsCount = try c.decodeIfPresent(Int.self, forKey: .participantsCount)
        mediaCount = try c.decodeIfPresent(Int.self, forKey: .mediaCount)
        startDateUnix = try c.decodeIfPresent(Int.self, forKey: .startDateUnix)
        endDateUnix = try c.decodeIfPresent(Int.self, forKey: .endDateUnix)
        createdAtUnix = try c.decode(Int.self, forKey: .createdAtUnix)
        updatedAtUnix = try c.decode(Int.self, forKey: .updatedAtUnix)
        isNameCensored = (try? c.decodeIfPresent(Bool.self, forKey: .isNameCensored)) ?? false
        isDescriptionCensored = (try? c.decodeIfPresent(Bool.self, forKey: .isDescriptionCensored)) ?? false
    }
}
