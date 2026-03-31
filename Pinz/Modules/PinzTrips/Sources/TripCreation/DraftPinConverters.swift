import PinzDomain

extension DraftPinDTO {
    func toRawPin() -> RawPin {
        RawPin(
            serverDraftPinId: draftPinId,
            medias: media.map { $0.toRawPinMedia() }
        )
    }
}

extension DraftPinMediaDTO {
    func toRawPinMedia() -> RawPinMedia {
        RawPinMedia(
            serverMediaId: mediaId,
            url: url,
            type: type.toMediaType()
        )
    }
}
