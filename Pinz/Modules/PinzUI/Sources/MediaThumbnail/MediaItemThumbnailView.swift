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
    private let showsPinMediaDeleteControl: Bool
    private let pinMediaDeleteBusy: Bool
    private let onPinMediaDelete: (() -> Void)?
    private let pinIdForServerMediaDelete: String?
    private let pinResponseAction: PinResponseAction?
    private let allowsMediaPrivacyChange: Bool

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
        onMediaUpdated: ((MediaItem) -> Void)? = nil,
        showsPinMediaDeleteControl: Bool = false,
        pinMediaDeleteBusy: Bool = false,
        onPinMediaDelete: (() -> Void)? = nil,
        pinIdForServerMediaDelete: String? = nil,
        pinResponseAction: PinResponseAction? = nil,
        allowsMediaPrivacyChange: Bool = true
    ) {
        self.mediaItem = mediaItem
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.hideBadges = hideBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.cacheVariant = cacheVariant
        self.cacheTargetPixel = cacheTargetPixel
        self.onMediaUpdated = onMediaUpdated
        self.showsPinMediaDeleteControl = showsPinMediaDeleteControl
        self.pinMediaDeleteBusy = pinMediaDeleteBusy
        self.onPinMediaDelete = onPinMediaDelete
        self.pinIdForServerMediaDelete = pinIdForServerMediaDelete
        self.pinResponseAction = pinResponseAction
        self.allowsMediaPrivacyChange = allowsMediaPrivacyChange
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
        .overlay {
            if showsPinMediaDeleteControl, onPinMediaDelete != nil {
                MediaBadgesView(trailingTopBadge: {
                    Button {
                        onPinMediaDelete?()
                    } label: {
                        BadgeView(icon: .trash, color: PinzUIAsset.accentRed.swiftUIColor)
                    }
                    .disabled(pinMediaDeleteBusy)
                    .opacity(pinMediaDeleteBusy ? 0.45 : 1)
                })
                .padding(4)
            }
        }
        .contextMenu {
            Button {
                if dismissBeforeMediaInfo { dismiss() }
                router?.navigateToMediaInfo(
                    media: mediaItem,
                    updateAction: onMediaUpdated.map { MediaUpdateAction(action: $0) },
                    pinIdForServerMediaDelete: pinIdForServerMediaDelete,
                    pinResponseAction: pinResponseAction,
                    allowsMediaPrivacyChange: allowsMediaPrivacyChange
                )
            } label: {
                Label(PinzBaseStrings.Common.Button.details, systemImage: "eye.fill")
            }

            if onPinMediaDelete != nil {
                Divider()
                Button(role: .destructive) {
                    onPinMediaDelete?()
                } label: {
                    Label(PinzBaseStrings.Common.Button.delete, systemImage: "trash")
                }
                .disabled(pinMediaDeleteBusy)
            }
        }
    }
}
