import Foundation
import CoreLocation

public struct RecommendedPinDTO: Codable {
    public let id: String
    public let tripId: String?
    public let name: String?
    public let description: String?
    public let category: String?
    public let latitude: Double?
    public let longitude: Double?
    public let locationName: String?
    public let mediaCount: Int?
    public let media: [FeedMediaDTO]?

    public init(
        id: String,
        tripId: String?,
        name: String?,
        description: String?,
        category: String?,
        latitude: Double?,
        longitude: Double?,
        locationName: String?,
        mediaCount: Int?,
        media: [FeedMediaDTO]?
    ) {
        self.id = id
        self.tripId = tripId
        self.name = name
        self.description = description
        self.category = category
        self.latitude = latitude
        self.longitude = longitude
        self.locationName = locationName
        self.mediaCount = mediaCount
        self.media = media
    }

    enum CodingKeys: String, CodingKey {
        case id
        case tripId = "trip_id"
        case name
        case description
        case category
        case latitude
        case longitude
        case locationName = "location_name"
        case mediaCount = "media_count"
        case media
    }

    public func mediaItems() -> [MediaItem] {
        media?.enumerated().compactMap { index, mediaItem in
            mediaItem.toMediaItem(id: index + 1)
        } ?? []
    }

    public func toPin(
        index: Int,
        medias: [MediaItem] = [],
        fallbackTripId: String? = nil,
        nameIfMissing: String
    ) -> Pin {
        let coordinates: CLLocationCoordinate2D?
        if let latitude, let longitude {
            coordinates = CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
        } else {
            coordinates = nil
        }

        let resolvedTripId = tripId ?? fallbackTripId
        let title = name?.isEmpty == false ? name! : nameIfMissing

        return Pin(
            name: title,
            description: description,
            category: category?.toPinCategory() ?? .custom(nil),
            medias: medias,
            isPrivate: false,
            startDate: nil,
            endDate: nil,
            tags: [],
            issues: [],
            serverId: id,
            tripId: resolvedTripId,
            coordinates: coordinates
        )
    }
}

