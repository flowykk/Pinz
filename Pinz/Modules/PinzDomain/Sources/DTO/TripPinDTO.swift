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
    public let isNameCensored: Bool
    public let isDescriptionCensored: Bool

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
        issues: [String]? = nil,
        isNameCensored: Bool = false,
        isDescriptionCensored: Bool = false
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
        self.isNameCensored = isNameCensored
        self.isDescriptionCensored = isDescriptionCensored
    }

    enum CodingKeys: String, CodingKey {
        case id, name, description, category, latitude, longitude, tags, media, issues
        case tripId = "trip_id"
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
        case privacyLevel = "privacy_level"
        case isNameCensored = "name_censored"
        case isDescriptionCensored = "description_censored"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        tripId = try c.decodeIfPresent(String.self, forKey: .tripId)
        name = try c.decodeIfPresent(String.self, forKey: .name)
        description = try c.decodeIfPresent(String.self, forKey: .description)
        category = try c.decodeIfPresent(String.self, forKey: .category)
        latitude = try c.decodeIfPresent(Double.self, forKey: .latitude)
        longitude = try c.decodeIfPresent(Double.self, forKey: .longitude)
        startTimeUnix = try c.decodeIfPresent(Int.self, forKey: .startTimeUnix)
        endTimeUnix = try c.decodeIfPresent(Int.self, forKey: .endTimeUnix)
        tags = try c.decodeIfPresent([String].self, forKey: .tags)
        privacyLevel = try c.decodeIfPresent(String.self, forKey: .privacyLevel)
        media = try c.decodeIfPresent([TripPinMediaDTO].self, forKey: .media)
        issues = try c.decodeIfPresent([String].self, forKey: .issues)
        isNameCensored = (try? c.decodeIfPresent(Bool.self, forKey: .isNameCensored)) ?? false
        isDescriptionCensored = (try? c.decodeIfPresent(Bool.self, forKey: .isDescriptionCensored)) ?? false
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
            coordinates: coordinates,
            isNameCensored: isNameCensored,
            isDescriptionCensored: isDescriptionCensored
        )
    }
}
