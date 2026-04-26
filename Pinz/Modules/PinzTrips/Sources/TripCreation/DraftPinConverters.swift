import CoreLocation
import PinzDomain
import PinzBase

extension DraftPinDTO {
    func toRawPin() -> RawPin {
        RawPin(
            id: draftPinId,
            medias: media.map { $0.toRawPinMedia() }
        )
    }
}

extension DraftPinMediaDTO {
    func toRawPinMedia() -> RawPinMedia {
        RawPinMedia(
            id: mediaId,
            url: url,
            type: type.toMediaType()
        )
    }
}

extension TripPinDTO {
    func toPin(index: Int, tripId fallbackTripId: String? = nil) -> Pin {
        let resolvedTripId = tripId ?? fallbackTripId
        let coordinates: CLLocationCoordinate2D?
        if let latitude, let longitude {
            coordinates = CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
        } else {
            coordinates = nil
        }

        let pinIsPrivate = privacyLevel?.lowercased() == "private"
        print("[toPin] pin \(id) privacyLevel=\(privacyLevel ?? "nil") → isPrivate=\(pinIsPrivate)")
        let mappedMedias: [MediaItem] = (media ?? []).enumerated().map { offset, m in
            let mediaIsPrivate = m.privacyLevel?.lowercased() == "private"
            print("[toPin]   media \(m.mediaId) privacyLevel=\(m.privacyLevel ?? "nil") → isPrivate=\(mediaIsPrivate)")
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
            name: name ?? PinzBaseStrings.Common.Label.pinNumber(index + 1),
            category: category?.toPinCategory() ?? .custom(nil),
            medias: mappedMedias,
            isPrivate: pinIsPrivate,
            startDate: startTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            endDate: endTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            tags: (tags ?? []).map { MediaTag(tag: $0) },
            issues: [],
            serverId: id,
            tripId: resolvedTripId,
            coordinates: coordinates
        )
    }
}

extension ReviewPinDTO {
    func toPin(index: Int, tripId: String? = nil) -> Pin {
        let coordinates: CLLocationCoordinate2D?
        if let latitude, let longitude {
            coordinates = CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
        } else {
            coordinates = nil
        }

        return Pin(
            name: name ?? PinzBaseStrings.Common.Label.pinNumber(index + 1),
            category: category?.toPinCategory() ?? .custom(nil),
            medias: (media ?? []).enumerated().map { offset, m in
                MediaItem(
                    id: offset + 1,
                    isPrivate: m.privacyLevel?.lowercased() == "private",
                    type: m.url.lowercased().contains(".mp4") || m.url.lowercased().contains(".mov") ? .video : .image,
                    mediaURL: URL(string: m.url),
                    tripId: tripId,
                    mediaId: m.mediaId
                )
            },
            isPrivate: false,
            startDate: startTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            endDate: endTimeUnix.map { Date(timeIntervalSince1970: Double($0)) },
            tags: (tags ?? []).map { MediaTag(tag: $0) },
            issues: issues ?? [],
            serverId: pinId,
            tripId: tripId,
            coordinates: coordinates
        )
    }
}

extension TripDTO {
    func toTrip() -> Trip {
        let season: TripSeason = {
            switch self.season?.lowercased() {
            case "summer": return .summer
            case "autumn", "fall": return .autumn
            case "winter": return .winter
            case "spring": return .spring
            default: return .none
            }
        }()

        let category: TripCategory = {
            switch self.category?.lowercased() {
            case "vacation": return .vacation
            case "holidays", "holiday": return .holidays
            case "business": return .business
            case "education": return .education
            case "active": return .active
            default: return self.category.map { .custom($0) } ?? .none
            }
        }()

        return Trip(
            id: id,
            name: name,
            description: description,
            pins: [],
            season: season,
            startDate: startDateUnix.map { Date(timeIntervalSince1970: Double($0)) },
            endDate: endDateUnix.map { Date(timeIntervalSince1970: Double($0)) },
            category: category,
            participantsCount: participantsCount ?? 0,
            mediaCount: mediaCount ?? 0,
            coverUrl: coverUrl,
            ownerUserId: ownerUserId,
            privacyLevel: privacyLevel,
            status: status,
            isPublished: isPublished,
            isGenerated: isGenerated,
            likesCount: likesCount,
            dislikesCount: dislikesCount,
            createdAt: Date(timeIntervalSince1970: Double(createdAtUnix)),
            updatedAt: Date(timeIntervalSince1970: Double(updatedAtUnix))
        )
    }
}
