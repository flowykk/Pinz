import Foundation

public struct VisitedLocationsResponseDTO: Codable {
    public let locations: [VisitedLocationDTO]

    public init(locations: [VisitedLocationDTO] = []) {
        self.locations = locations
    }

    enum CodingKeys: String, CodingKey {
        case locations
        case data
        case items
    }

    public init(from decoder: Decoder) throws {
        if let direct = try? decoder.singleValueContainer(),
           let directLocations = try? direct.decode([VisitedLocationDTO].self) {
            self.locations = directLocations
            return
        }

        guard let container = try? decoder.container(keyedBy: CodingKeys.self) else {
            self.locations = []
            return
        }

        self.locations = (try? container.decodeIfPresent([VisitedLocationDTO].self, forKey: .locations))
            ?? (try? container.decodeIfPresent([VisitedLocationDTO].self, forKey: .data))
            ?? (try? container.decodeIfPresent([VisitedLocationDTO].self, forKey: .items))
            ?? []
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(locations, forKey: .locations)
    }
}
