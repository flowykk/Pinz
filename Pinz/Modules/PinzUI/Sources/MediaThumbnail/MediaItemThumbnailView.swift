import SwiftUI
import PinzBase
import PinzDomain

public struct MediaItemThumbnailView: View {

    private let mediaItem: MediaItem
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let hideBadges: Bool
    private let dismissBeforeMediaInfo: Bool
    private let cacheVariant: MediaCacheVariant
    private let cacheTargetPixel: Int
    private let onMediaUpdated: ((MediaItem) -> Void)?

    @Environment(\.dismiss) private var dismiss
    @Environment(\.appRouter) private var router

    public init(
        mediaItem: MediaItem,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        hideBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        cacheVariant: MediaCacheVariant = .thumbnail,
        cacheTargetPixel: Int = 560,
        onMediaUpdated: ((MediaItem) -> Void)? = nil
    ) {
        self.mediaItem = mediaItem
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.hideBadges = hideBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.cacheVariant = cacheVariant
        self.cacheTargetPixel = cacheTargetPixel
        self.onMediaUpdated = onMediaUpdated
    }

    public var body: some View {
        MediaThumbnailView(
            url: mediaItem.mediaURL,
            type: mediaItem.type,
            contentMode: contentMode,
            cornerRadius: cornerRadius,
            cacheVariant: cacheVariant,
            cacheTargetPixel: cacheTargetPixel
        )
        .if(!hideBadges) { view in
            view.overlay { PrivateVideoMediaBadgesView(media: mediaItem).padding(4) }
        }
        .contextMenu {
            Button {
                if dismissBeforeMediaInfo { dismiss() }
                router?.navigateToMediaInfo(media: mediaItem, updateAction: onMediaUpdated.map { MediaUpdateAction(action: $0) })
            } label: {
                Label(PinzBaseStrings.Common.Button.details, systemImage: "eye.fill")
            }

            Divider()
            Button(role: .destructive) {

            } label: {
                Label(PinzBaseStrings.Common.Button.delete, systemImage: "trash")
            }
        }
    }
}
