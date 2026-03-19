public struct JoinTripByTokenDTO: Codable {
    public let tripId: String
    public let alreadyJoined: Bool

    public init(tripId: String, alreadyJoined: Bool) {
        self.tripId = tripId
        self.alreadyJoined = alreadyJoined
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case alreadyJoined = "already_joined"
    }
}
