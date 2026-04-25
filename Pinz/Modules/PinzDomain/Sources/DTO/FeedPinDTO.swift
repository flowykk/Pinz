import Foundation
import CoreLocation

public struct FeedPinDTO: Codable {
    public let id: String
    public let latitude: Double
    public let longitude: Double
    public let media: [FeedMediaDTO]?

    public init(
        id: String,
        latitude: Double,
        longitude: Double,
        media: [FeedMediaDTO]? = nil
    ) {
        self.id = id
        self.latitude = latitude
        self.longitude = longitude
        self.media = media
    }

    public func toPin(index: Int, medias: [MediaItem] = []) -> Pin {
        Pin(
            name: "Pin \(index + 1)",
            category: .custom(nil),
            medias: medias,
            isPrivate: false,
            tags: [],
            issues: [],
            serverId: id,
            coordinates: CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
        )
    }

    public func mediaItems() -> [MediaItem] {
        media?.enumerated().compactMap { index, mediaItem in
            mediaItem.toMediaItem(id: index + 1)
        } ?? []
    }
}
