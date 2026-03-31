import SwiftUI

public enum MediaType: Codable, Hashable {
    case image
    case video

    public var contentType: String {
        switch self {
        case .image: return "image/jpeg"
        case .video: return "video/quicktime"
        }
    }

    public var rawValue: String {
        switch self {
        case .image: return "image"
        case .video: return "video"
        }
    }
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
