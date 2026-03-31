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
    public var photosPickerItem: PhotosPickerItem?
    public var coordinates: MediaCoordinates?
    public var videoEditingSettings: VideoEditingSettings?

    public init(
        id: UUID = UUID(),
        content: Content,
        photosPickerItem: PhotosPickerItem? = nil,
        coordinates: MediaCoordinates? = nil,
        videoEditingSettings: VideoEditingSettings? = nil
    ) {
        self.id = id
        self.content = content
        self.photosPickerItem = photosPickerItem
        self.coordinates = coordinates
        self.videoEditingSettings = videoEditingSettings
    }
}

extension LoadedMedia {
    public func uploadData() async -> Data? {
        switch content {
        case .image(let image):
            return image.jpegData(compressionQuality: 0.85)
        case .video(let url, _):
            return await Task.detached { try? Data(contentsOf: url) }.value
        case .loading:
            return nil
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
