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

public struct MediaCoordinates: Hashable {
    public let latitude: Double
    public let longitude: Double

    public init(latitude: Double, longitude: Double) {
        self.latitude = latitude
        self.longitude = longitude
    }
}
