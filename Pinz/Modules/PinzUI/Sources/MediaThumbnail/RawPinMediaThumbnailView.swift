import SwiftUI
import PinzDomain

public struct RawPinMediaThumbnailView: View {

    private let media: RawPinMedia
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let square: Bool

    public init(
        media: RawPinMedia,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        square: Bool = false
    ) {
        self.media = media
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.square = square
    }

    public var body: some View {
        MediaThumbnailView(
            url: URL(string: media.url),
            type: media.type,
            contentMode: contentMode,
            cornerRadius: cornerRadius,
            square: square
        )
    }
}
