import Foundation

public struct UserStatsResponseDTO: Codable {
    public let totalTrips: Int?
    public let totalPins: Int?
    public let totalMedia: Int?
    public let totalLikes: Int?
    public let totalDislikes: Int?
    public let battlesFinished: Int?

    public init(
        totalTrips: Int? = nil,
        totalPins: Int? = nil,
        totalMedia: Int? = nil,
        totalLikes: Int? = nil,
        totalDislikes: Int? = nil,
        battlesFinished: Int? = nil
    ) {
        self.totalTrips = totalTrips
        self.totalPins = totalPins
        self.totalMedia = totalMedia
        self.totalLikes = totalLikes
        self.totalDislikes = totalDislikes
        self.battlesFinished = battlesFinished
    }

    enum CodingKeys: String, CodingKey {
        case totalTrips = "total_trips"
        case totalPins = "total_pins"
        case totalMedia = "total_media"
        case totalLikes = "total_likes"
        case totalDislikes = "total_dislikes"
        case battlesFinished = "battles_finished"
    }
}
