import PinzDomain

// MARK: - Internal Encodable wrappers mapping DTOs to snake_case JSON

struct FileToUploadJSON: Encodable {
    let client_id: String
    let content_type: String

    init(_ dto: FileToUploadDTO) {
        client_id = dto.clientId
        content_type = dto.contentType
    }
}

struct MediaMetaEntryJSON: Encodable {
    let s3_key: String
    let captured_at: String?
    let latitude: Double?
    let longitude: Double?
    let media_type: String
    let content_hash: String?

    init(_ dto: MediaMetaEntryDTO) {
        s3_key = dto.s3Key
        captured_at = dto.capturedAt
        latitude = dto.latitude
        longitude = dto.longitude
        media_type = dto.mediaType
        content_hash = dto.contentHash
    }
}

struct DraftPinInputJSON: Encodable {
    let draft_pin_id: String
    let media_ids: [String]

    init(_ dto: DraftPinInputDTO) {
        draft_pin_id = dto.draftPinId
        media_ids = dto.mediaIds
    }
}

struct PinUpdateInputJSON: Encodable {
    let pin_id: String
    let name: String?
    let description: String?
    let category: String?
    let privacy_level: String?
    let latitude: Double?
    let longitude: Double?
    let tags: [String]?
    let start_time_unix: Int?
    let end_time_unix: Int?

    init(_ dto: PinUpdateInputDTO) {
        pin_id = dto.pinId
        name = dto.name
        description = dto.description
        category = dto.category
        privacy_level = dto.privacyLevel
        latitude = dto.latitude
        longitude = dto.longitude
        tags = dto.tags
        start_time_unix = dto.startTimeUnix
        end_time_unix = dto.endTimeUnix
    }
}
