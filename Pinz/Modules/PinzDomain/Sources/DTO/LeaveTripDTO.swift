public struct LeaveTripDTO: Codable {
    public let success: Bool
    public let tripDeleted: Bool

    public init(success: Bool, tripDeleted: Bool) {
        self.success = success
        self.tripDeleted = tripDeleted
    }

    enum CodingKeys: String, CodingKey {
        case success
        case tripDeleted = "trip_deleted"
    }
}
