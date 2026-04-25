import Foundation

public struct FeedMediaDTO: Codable {
    public let mediaId: String
    public let url: String
    public let mediaType: String

    public init(
        mediaId: String,
        url: String,
        mediaType: String
    ) {
        self.mediaId = mediaId
        self.url = url
        self.mediaType = mediaType
    }

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case url
        case mediaType = "media_type"
    }

    public func toMediaItem(id: Int) -> MediaItem? {
        guard let url = URL(string: url) else { return nil }
        return MediaItem(
            id: id,
            isPrivate: false,
            type: mediaType.toMediaType(),
            mediaURL: url
        )
    }
}
