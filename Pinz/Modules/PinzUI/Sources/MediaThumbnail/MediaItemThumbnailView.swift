import SwiftUI
import PinzBase
import PinzDomain

public struct MediaItemThumbnailView: View {

    private let mediaItem: MediaItem
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let hideBadges: Bool
    private let dismissBeforeMediaInfo: Bool

    @Environment(\.dismiss) private var dismiss
    @Environment(\.appRouter) private var router

    public init(
        mediaItem: MediaItem,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        hideBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false
    ) {
        self.mediaItem = mediaItem
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.hideBadges = hideBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
    }

    public var body: some View {
        MediaThumbnailView(
            url: mediaItem.mediaURL,
            type: mediaItem.type,
            contentMode: contentMode,
            cornerRadius: cornerRadius
        )
        .if(!hideBadges) { view in
            view.overlay { PrivateVideoMediaBadgesView(media: mediaItem).padding(4) }
        }
        .contextMenu {
            Button {
                if dismissBeforeMediaInfo { dismiss() }
                router?.navigateToMediaInfo(media: mediaItem)
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
