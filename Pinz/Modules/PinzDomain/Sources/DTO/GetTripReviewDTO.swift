public struct GetTripReviewDTO: Codable {
    public let tripId: String
    public let status: String
    public let pins: [ReviewPinDTO]
    public let similar: [[String]]

    public init(tripId: String, status: String, pins: [ReviewPinDTO], similar: [[String]]) {
        self.tripId = tripId
        self.status = status
        self.pins = pins
        self.similar = similar
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case status, pins, similar
    }
}
