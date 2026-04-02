import CoreLocation
import PinzDomain

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
            name: name ?? "Пин \(index + 1)",
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
