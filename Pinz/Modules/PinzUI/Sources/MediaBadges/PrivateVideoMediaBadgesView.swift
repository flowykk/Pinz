import SwiftUI
import PinzDomain

public struct PrivateVideoMediaBadgesView: View {

    private let media: MediaItem

    public init(media: MediaItem) {
        self.media = media
    }

    public var body: some View {
        MediaBadgesView(leadingTopBadge: {
            VStack(spacing: 4) {
                BadgeView(
                    icon: media.isPrivate ? .lock : .lockOpen,
                    color: media.isPrivate ? PinzUIAsset.accentRed.swiftUIColor : .white
                )
                if media.type == .video { BadgeView(icon: .video) }
            }
        })
    }
}
