public struct DraftPinInputDTO {
    public let draftPinId: String
    public let mediaIds: [String]

    public init(draftPinId: String, mediaIds: [String]) {
        self.draftPinId = draftPinId
        self.mediaIds = mediaIds
    }
}
