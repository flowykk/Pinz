import Foundation

public struct GetTripResponseDTO: Codable {
    public let trip: TripDTO
    public let pins: [ReviewPinDTO]

    enum CodingKeys: String, CodingKey {
        case trip
        case pins
    }

    public init(trip: TripDTO, pins: [ReviewPinDTO]) {
        self.trip = trip
        self.pins = pins
    }
}
