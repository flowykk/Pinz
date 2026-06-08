import SwiftUI
import PhotosUI

public struct LoadedMedia: Hashable, Identifiable {
    public enum Content: Hashable, Sendable {
        case loading
        case image(UIImage)
        case video(url: URL, firstFrame: UIImage)
    }

    public let id: UUID
    public var content: Content
    public var imageFileData: Data?
    public var contentType: String?
    public var photosPickerItem: PhotosPickerItem?
    public var coordinates: MediaCoordinates?
    public var capturedAt: String?
    public var contentHash: String?
    public var videoEditingSettings: VideoEditingSettings?

    public init(
        id: UUID = UUID(),
        content: Content,
        imageFileData: Data? = nil,
        contentType: String? = nil,
        photosPickerItem: PhotosPickerItem? = nil,
        coordinates: MediaCoordinates? = nil,
        capturedAt: String? = nil,
        contentHash: String? = nil,
        videoEditingSettings: VideoEditingSettings? = nil
    ) {
        self.id = id
        self.content = content
        self.imageFileData = imageFileData
        self.contentType = contentType
        self.photosPickerItem = photosPickerItem
        self.coordinates = coordinates
        self.capturedAt = capturedAt
        self.contentHash = contentHash
        self.videoEditingSettings = videoEditingSettings
    }
}

extension LoadedMedia {
    public func uploadData() async -> Data? {
        switch content {
        case .image:
            return imageFileData
        case .video(let url, _):
            return await Task.detached { try? Data(contentsOf: url) }.value
        case .loading:
            return nil
        }
    }

    public var uploadContentType: String {
        switch content {
        case .image:
            return contentType ?? MediaType.image.contentType
        case .video:
            return contentType ?? MediaType.video.contentType
        case .loading:
            return MediaType.image.contentType
        }
    }

    public var mediaType: MediaType {
        switch content {
        case .image: return .image
        case .video: return .video
        case .loading: return .image
        }
    }
}

public struct MediaCoordinates: Hashable {
    public let latitude: Double
    public let longitude: Double

    public init(latitude: Double, longitude: Double) {
        self.latitude = latitude
        self.longitude = longitude
    }
}
