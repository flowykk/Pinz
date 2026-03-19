public struct FinalizeTripDTO: Codable {
    public let tripId: String
    public let status: String
    public let message: String

    public init(tripId: String, status: String, message: String) {
        self.tripId = tripId
        self.status = status
        self.message = message
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case status, message
    }
}
