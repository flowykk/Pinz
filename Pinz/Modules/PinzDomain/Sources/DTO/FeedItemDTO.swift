import Foundation

public struct FeedItemDTO: Codable {
    public let trip: TripDTO
    public let pins: [FeedPinDTO]
    public let media: [FeedMediaDTO]
    public let isLiked: Bool
    public let isDisliked: Bool
    public let isSaved: Bool

    public init(
        trip: TripDTO,
        pins: [FeedPinDTO],
        media: [FeedMediaDTO],
        isLiked: Bool = false,
        isDisliked: Bool = false,
        isSaved: Bool = false
    ) {
        self.trip = trip
        self.pins = pins
        self.media = media
        self.isLiked = isLiked
        self.isDisliked = isDisliked
        self.isSaved = isSaved
    }

    enum CodingKeys: String, CodingKey {
        case trip
        case pins
        case media
        case isLiked = "is_liked"
        case isDisliked = "is_disliked"
        case isSaved = "is_saved"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        trip = try container.decode(TripDTO.self, forKey: .trip)
        pins = try container.decodeIfPresent([FeedPinDTO].self, forKey: .pins) ?? []
        media = try container.decodeIfPresent([FeedMediaDTO].self, forKey: .media) ?? []
        isLiked = try container.decodeIfPresent(Bool.self, forKey: .isLiked) ?? false
        isDisliked = try container.decodeIfPresent(Bool.self, forKey: .isDisliked) ?? false
        isSaved = try container.decodeIfPresent(Bool.self, forKey: .isSaved) ?? false
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(trip, forKey: .trip)
        try container.encode(pins, forKey: .pins)
        try container.encode(media, forKey: .media)
        try container.encode(isLiked, forKey: .isLiked)
        try container.encode(isDisliked, forKey: .isDisliked)
        try container.encode(isSaved, forKey: .isSaved)
    }
}
