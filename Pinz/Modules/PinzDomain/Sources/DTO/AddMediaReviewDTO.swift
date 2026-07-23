public struct AddMediaReviewDTO: Codable {
    public let tripId: String
    public let sessionId: String
    public let pins: [TripPinDTO]
    public let newPinIds: [String]
    public let protectedMediaIds: [String]
    public let currentInitiator: PublicUserProfileDTO?
    public let takeoverAvailableAt: String?
    public let canEdit: Bool

    public init(
        tripId: String,
        sessionId: String,
        pins: [TripPinDTO],
        newPinIds: [String],
        protectedMediaIds: [String],
        currentInitiator: PublicUserProfileDTO? = nil,
        takeoverAvailableAt: String? = nil,
        canEdit: Bool
    ) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.pins = pins
        self.newPinIds = newPinIds
        self.protectedMediaIds = protectedMediaIds
        self.currentInitiator = currentInitiator
        self.takeoverAvailableAt = takeoverAvailableAt
        self.canEdit = canEdit
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case sessionId = "session_id"
        case pins
        case newPinIds = "new_pin_ids"
        case protectedMediaIds = "protected_media_ids"
        case currentInitiator = "current_initiator"
        case takeoverAvailableAt = "takeover_available_at"
        case canEdit = "can_edit"
    }
}
