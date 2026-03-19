public struct MediaMetaEntryDTO {
    public let s3Key: String
    public let capturedAt: String?
    public let latitude: Double?
    public let longitude: Double?
    public let mediaType: String
    public let contentHash: String?

    public init(
        s3Key: String,
        capturedAt: String?,
        latitude: Double?,
        longitude: Double?,
        mediaType: String,
        contentHash: String?
    ) {
        self.s3Key = s3Key
        self.capturedAt = capturedAt
        self.latitude = latitude
        self.longitude = longitude
        self.mediaType = mediaType
        self.contentHash = contentHash
    }
}
