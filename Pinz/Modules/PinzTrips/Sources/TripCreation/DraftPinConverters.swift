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
