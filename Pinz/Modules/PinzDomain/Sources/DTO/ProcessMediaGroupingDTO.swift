public struct ProcessMediaGroupingDTO: Codable {
    public let tripId: String
    public let status: String
    public let draftPins: [DraftPinDTO]

    public init(tripId: String, status: String, draftPins: [DraftPinDTO]) {
        self.tripId = tripId
        self.status = status
        self.draftPins = draftPins
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case status
        case draftPins = "draft_pins"
    }
}
