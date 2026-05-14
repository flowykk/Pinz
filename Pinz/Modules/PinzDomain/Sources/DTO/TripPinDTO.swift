import Foundation
import CoreLocation

public struct TripPinDTO: Codable {
    public let id: String
    public let tripId: String?
    public let name: String?
    public let description: String?
    public let category: String?
    public let latitude: Double?
    public let longitude: Double?
    public let startTimeUnix: Int?
    public let endTimeUnix: Int?
    public let tags: [String]?
    public let privacyLevel: String?
    public let media: [TripPinMediaDTO]?
    public let issues: [String]?

    public init(
        id: String,
        tripId: String?,
        name: String?,
        description: String?,
        category: String?,
        latitude: Double?,
        longitude: Double?,
        startTimeUnix: Int?,
        endTimeUnix: Int?,
        tags: [String]?,
        privacyLevel: String?,
        media: [TripPinMediaDTO]?,
        issues: [String]? = nil
    ) {
        self.id = id
        self.tripId = tripId
        self.name = name
        self.description = description
        self.category = category
        self.latitude = latitude
        self.longitude = longitude
        self.startTimeUnix = startTimeUnix
        self.endTimeUnix = endTimeUnix
        self.tags = tags
        self.privacyLevel = privacyLevel
        self.media = media
        self.issues = issues
    }

    enum CodingKeys: String, CodingKey {
        case id, name, description, category, latitude, longitude, tags, media, issues
        case tripId = "trip_id"
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
        case privacyLevel = "privacy_level"
    }
}

public extension TripPinDTO {
    func toPin(index: Int, tripId fallbackTripId: String? = nil, nameIfMissing: String) -> Pin {
        let resolvedTripId = tripId ?? fallbackTripId
        let coordinates: CLLocationCoordinate2D?
        if let latitude, let longitude {
            coordinates = CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
        } else {
            coordinates = nil
        }

        let title: String
        if let n = name, !n.isEmpty {
            title = n
        } else {
            title = nameIfMissing
        }

        let pinIsPrivate = privacyLevel?.lowercased() == "private"
        let mappedMedias: [MediaItem] = (media ?? []).enumerated().map { offset, m in
            let mediaIsPrivate = m.privacyLevel?.lowercased() == "private"
            return MediaItem(
                id: offset + 1,
                isPrivate: mediaIsPrivate,
                type: m.mediaType == "video" ? .video : .image,
                mediaURL: URL(string: m.url),
                tripId: resolvedTripId,
                mediaId: m.mediaId
            )
        }
        return Pin(
            name: title,
            description: description,
            category: category?.toPinCategory() ?? .custom(nil),
            medias: mappedMedias,
            isPrivate: pinIsPrivate,
            startDate: startTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            endDate: endTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            tags: (tags ?? []).map { MediaTag(tag: $0) },
            issues: issues ?? [],
            serverId: id,
            tripId: resolvedTripId,
            coordinates: coordinates
        )
    }
}
