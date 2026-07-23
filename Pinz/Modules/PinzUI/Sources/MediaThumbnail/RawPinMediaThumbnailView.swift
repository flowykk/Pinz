import SwiftUI
import PinzBase
import PinzDomain

public struct RawPinMediaThumbnailView: View {

    private let media: RawPinMedia
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let square: Bool
    private let cacheVariant: MediaCacheVariant
    private let cacheTargetPixel: Int

    public init(
        media: RawPinMedia,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        square: Bool = false,
        cacheVariant: MediaCacheVariant = .thumbnail,
        cacheTargetPixel: Int = 560
    ) {
        self.media = media
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.square = square
        self.cacheVariant = cacheVariant
        self.cacheTargetPixel = cacheTargetPixel
    }

    public var body: some View {
        MediaThumbnailView(
            url: URL(string: media.url),
            type: media.type,
            contentMode: contentMode,
            cornerRadius: cornerRadius,
            square: square,
            cacheVariant: cacheVariant,
            cacheTargetPixel: cacheTargetPixel
        )
    }
}
