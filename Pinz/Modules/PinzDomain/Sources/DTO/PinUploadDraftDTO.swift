public struct PinUploadDraftDTO: Codable {
    public let suggested: PinSuggestedFieldsDTO?
    public let media: [ReviewPinMediaDTO]?
    public let pinIssues: [String]?
    public let nsfwMediaIds: [String]?
    public let dedupedMediaIds: [String]?

    public init(
        suggested: PinSuggestedFieldsDTO?,
        media: [ReviewPinMediaDTO]?,
        pinIssues: [String]?,
        nsfwMediaIds: [String]?,
        dedupedMediaIds: [String]?
    ) {
        self.suggested = suggested
        self.media = media
        self.pinIssues = pinIssues
        self.nsfwMediaIds = nsfwMediaIds
        self.dedupedMediaIds = dedupedMediaIds
    }

    enum CodingKeys: String, CodingKey {
        case suggested, media
        case pinIssues = "pin_issues"
        case nsfwMediaIds = "nsfw_media_ids"
        case dedupedMediaIds = "deduped_media_ids"
    }
}
