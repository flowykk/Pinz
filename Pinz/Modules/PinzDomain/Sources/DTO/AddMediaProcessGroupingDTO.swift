import Foundation

public struct AddMediaProcessGroupingDTO: Codable {
    public let tripId: String
    public let status: String
    public let draftPins: [DraftPinDTO]
    public let existingMediaIds: [String]?

    public init(
        tripId: String,
        status: String,
        draftPins: [DraftPinDTO],
        existingMediaIds: [String]?
    ) {
        self.tripId = tripId
        self.status = status
        self.draftPins = draftPins
        self.existingMediaIds = existingMediaIds
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case status
        case draftPins = "draft_pins"
        case existingMediaIds = "existing_media_ids"
    }
}
