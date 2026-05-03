public struct AddMediaGroupingDTO: Codable {
    public let tripId: String
    public let sessionId: String
    public let status: String?
    public let draftPins: [DraftPinDTO]
    public let existingMediaIds: [String]?

    public init(
        tripId: String,
        sessionId: String,
        status: String? = nil,
        draftPins: [DraftPinDTO],
        existingMediaIds: [String]? = nil
    ) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.status = status
        self.draftPins = draftPins
        self.existingMediaIds = existingMediaIds
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case sessionId = "session_id"
        case status
        case draftPins = "draft_pins"
        case existingMediaIds = "existing_media_ids"
    }
}
