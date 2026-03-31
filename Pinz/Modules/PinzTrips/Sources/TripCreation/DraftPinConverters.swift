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
