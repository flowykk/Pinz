import Foundation

public struct FeedItemDTO: Codable {
    public let trip: TripDTO
    public let pins: [FeedPinDTO]
    public let media: [FeedMediaDTO]

    public init(
        trip: TripDTO,
        pins: [FeedPinDTO],
        media: [FeedMediaDTO]
    ) {
        self.trip = trip
        self.pins = pins
        self.media = media
    }

    enum CodingKeys: String, CodingKey {
        case trip
        case pins
        case media
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        trip = try container.decode(TripDTO.self, forKey: .trip)
        pins = try container.decodeIfPresent([FeedPinDTO].self, forKey: .pins) ?? []
        media = try container.decodeIfPresent([FeedMediaDTO].self, forKey: .media) ?? []
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(trip, forKey: .trip)
        try container.encode(pins, forKey: .pins)
        try container.encode(media, forKey: .media)
    }
}
