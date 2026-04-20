import Foundation

public struct UserStatsResponseDTO: Codable {
    public let tripsCount: Int?
    public let pinsCount: Int?
    public let mediaCount: Int?
    public let likesCount: Int?
    public let dislikesCount: Int?
    public let battlesCount: Int?

    public init(
        tripsCount: Int? = nil,
        pinsCount: Int? = nil,
        mediaCount: Int? = nil,
        likesCount: Int? = nil,
        dislikesCount: Int? = nil,
        battlesCount: Int? = nil
    ) {
        self.tripsCount = tripsCount
        self.pinsCount = pinsCount
        self.mediaCount = mediaCount
        self.likesCount = likesCount
        self.dislikesCount = dislikesCount
        self.battlesCount = battlesCount
    }

    enum CodingKeys: String, CodingKey {
        case tripsCount = "trips_count"
        case pinsCount = "pins_count"
        case mediaCount = "media_count"
        case likesCount = "likes_count"
        case dislikesCount = "dislikes_count"
        case battlesCount = "battles_count"
    }
}
