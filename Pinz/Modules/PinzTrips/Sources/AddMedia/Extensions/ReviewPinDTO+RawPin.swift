import PinzDomain

extension ReviewPinMediaDTO {
    func toRawPinMedia() -> RawPinMedia {
        let lowercasedURL = url.lowercased()
        let mediaType: MediaType = {
            if lowercasedURL.contains(".mp4") || lowercasedURL.contains(".mov") || lowercasedURL.contains(".m4v") {
                return .video
            }
            return .image
        }()

        return RawPinMedia(
            id: mediaId,
            url: url,
            type: mediaType
        )
    }
}

extension ReviewPinDTO {
    func toRawPin() -> RawPin {
        RawPin(
            id: id,
            medias: (media ?? []).map { $0.toRawPinMedia() }
        )
    }
}
