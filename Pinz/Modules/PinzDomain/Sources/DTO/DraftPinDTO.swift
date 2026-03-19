public struct DraftPinDTO: Codable {
    public let draftPinId: String
    public let media: [DraftPinMediaDTO]

    public init(draftPinId: String, media: [DraftPinMediaDTO]) {
        self.draftPinId = draftPinId
        self.media = media
    }

    enum CodingKeys: String, CodingKey {
        case draftPinId = "draft_pin_id"
        case media
    }
}
