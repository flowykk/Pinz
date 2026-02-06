import SwiftUI

public enum MediaType: Codable {
    case image
    case video
    case quote
}

public struct MediaItem: Identifiable, Codable, Hashable {
    public let id: Int
    public let mediaType: MediaType
    public let mediaURL: URL?

    public init(
        id: Int = -1,
        mediaType: MediaType,
        mediaURL: URL?
    ) {
        self.id = id
        self.mediaType = mediaType
        self.mediaURL = mediaURL
    }
}

extension MediaItem {
    public static func stubs() -> [MediaItem] {
        [
            MediaItem(
                mediaType: .image,
                mediaURL: URL(string: "https://pushinka.top/uploads/posts/2023-08/1692812086_pushinka-top-p-smekh-kartinki-smeshnie-vkontakte-13.jpg"),
            ),
            MediaItem(
                mediaType: .image,
                mediaURL: URL(string: "https://i.pinimg.com/originals/54/93/42/54934276d19ad7a7dc396dc8069bc485.jpg"),
            ),
            MediaItem(
                mediaType: .image,
                mediaURL: URL(string: "https://i2-prod.dailyrecord.co.uk/incoming/article1906467.ece/ALTERNATES/s1227b/laughing-animals.jpg"),
            ),
            MediaItem(
                mediaType: .image,
                mediaURL: URL(string: "https://i.pinimg.com/736x/80/09/ca/8009ca0a8bb73d596838a57d4c8fa491.jpg"),
            ),
            MediaItem(
                mediaType: .video,
                mediaURL: URL(string: "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"),
            )
        ]
    }
}

extension String {
    public func toMediaType() -> MediaType {
        switch self {
        case "image": .image
        case "video": .video
        default: .image
        }
    }
}
