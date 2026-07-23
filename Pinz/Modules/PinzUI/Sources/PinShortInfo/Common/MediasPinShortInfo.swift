//
//  MediasPinShortInfo.swift
//  PinzUI
//
//  Created by Danila Rakhmanov on 08.02.2026.
//

import SwiftUI
import PinzDomain
import PinzBase

public struct MediasPinShortInfo: View {
    let pin: Pin
    let maxMedias: Int
    let selectable: Bool
    let hideMediaBadges: Bool
    let dismissBeforeMediaInfo: Bool
    let allowsMediaPrivacyChange: Bool
    let onMediaUpdated: ((MediaItem) -> Void)?

    public init(
        pin: Pin,
        maxMedias: Int = 6,
        selectable: Bool = false,
        hideMediaBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        allowsMediaPrivacyChange: Bool = true,
        onMediaUpdated: ((MediaItem) -> Void)? = nil
    ) {
        self.pin = pin
        self.maxMedias = maxMedias
        self.selectable = selectable
        self.hideMediaBadges = hideMediaBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.allowsMediaPrivacyChange = allowsMediaPrivacyChange
        self.onMediaUpdated = onMediaUpdated
    }
    
    public var body: some View {
        let medias = selectable ? pin.medias.sorted { !$0.isPrivate && $1.isPrivate } : pin.medias
        
        ScrollView(.horizontal) {
            HStack(spacing: 4) {
                ForEach(medias.prefix(maxMedias)) { media in
                    MediaItemThumbnailView(
                        mediaItem: media,
                        contentMode: .fit,
                        cornerRadius: 14,
                        hideBadges: selectable || hideMediaBadges,
                        dismissBeforeMediaInfo: dismissBeforeMediaInfo,
                        onMediaUpdated: onMediaUpdated,
                        allowsMediaPrivacyChange: allowsMediaPrivacyChange
                    )
                    .frame(height: 96)
                    .opacity(selectable && media.isPrivate ? 0.5 : 1)
                    .overlay {
                        if !hideMediaBadges && selectable {
                            PrivateVideoMediaBadgesView(media: media).padding(4)
                        }
                    }
                }

                if medias.count > maxMedias {
                    VStack {
                        Text("+\(medias.count - maxMedias)")
                            .roundedFont(size: 24, weight: .semibold, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        Text(PinzBaseStrings.Common.Label.media)
                            .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    }.frame(76)
                }
            }.padding(.horizontal, 12)
        }.scrollIndicators(.hidden)
    }
}
