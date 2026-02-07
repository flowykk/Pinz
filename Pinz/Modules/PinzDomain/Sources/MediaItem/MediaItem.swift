import SwiftUI

public enum MediaType: Codable {
    case image
    case video
}

public struct MediaItem: Identifiable, Codable, Hashable {
    public let id: Int
    public let isPrivate: Bool
    public let type: MediaType
    public let mediaURL: URL?

    public init(
        id: Int = -1,
        isPrivate: Bool,
        type: MediaType,
        mediaURL: URL?
    ) {
        self.id = id
        self.isPrivate = isPrivate
        self.type = type
        self.mediaURL = mediaURL
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
