import Foundation

public struct RecommendedMapDTO: Codable {
    public let media: [FeedMediaDTO]?
    public let pins: [RecommendedPinDTO]
    public let regionName: String?
    public let regionType: String?
    public let trip: TripDTO

    public init(
        media: [FeedMediaDTO]?,
        pins: [RecommendedPinDTO],
        regionName: String?,
        regionType: String?,
        trip: TripDTO
    ) {
        self.media = media
        self.pins = pins
        self.regionName = regionName
        self.regionType = regionType
        self.trip = trip
    }

    enum CodingKeys: String, CodingKey {
        case media
        case pins
        case regionName = "region_name"
        case regionType = "region_type"
        case trip
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        media = try container.decodeIfPresent([FeedMediaDTO].self, forKey: .media)
        pins = try container.decodeIfPresent([RecommendedPinDTO].self, forKey: .pins) ?? []
        regionName = try container.decodeIfPresent(String.self, forKey: .regionName)
        regionType = try container.decodeIfPresent(String.self, forKey: .regionType)
        trip = try container.decode(TripDTO.self, forKey: .trip)
    }
}

