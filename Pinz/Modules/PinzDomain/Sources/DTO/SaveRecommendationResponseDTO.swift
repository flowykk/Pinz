import Foundation

public struct SaveRecommendationResponseDTO: Codable {
    public let trip: TripDTO

    public init(trip: TripDTO) {
        self.trip = trip
    }
}

