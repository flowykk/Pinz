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

extension ReviewPinDTO {
    func toPin(index: Int) -> Pin {
        Pin(
            name: name ?? PinzBaseStrings.Common.Label.pinNumber(index + 1),
            category: category?.toPinCategory() ?? .custom(nil),
            medias: media.enumerated().map { offset, m in
                MediaItem(
                    id: offset + 1,
                    isPrivate: m.privacyLevel != nil && m.privacyLevel != "public",
                    type: m.url.lowercased().contains(".mp4") || m.url.lowercased().contains(".mov") ? .video : .image,
                    mediaURL: URL(string: m.url)
                )
            },
            isPrivate: false,
            tags: tags.map { MediaTag(tag: $0) },
            coordinates: CLLocationCoordinate2D(
                latitude: latitude ?? 0,
                longitude: longitude ?? 0
            )
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
            members: [],
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
