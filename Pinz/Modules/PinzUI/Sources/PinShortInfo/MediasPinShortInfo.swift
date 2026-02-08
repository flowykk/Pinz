//
//  MediasPinShortInfo.swift
//  PinzUI
//
//  Created by Danila Rakhmanov on 08.02.2026.
//

import SwiftUI
import PinzDomain

public struct MediasPinShortInfo: View {
    let pin: Pin
    let maxMedias: Int
    let selectable: Bool
    let dismissBeforeMediaInfo: Bool

    public init(
        pin: Pin,
        maxMedias: Int = 6,
        selectable: Bool = false,
        dismissBeforeMediaInfo: Bool = false
    ) {
        self.pin = pin
        self.maxMedias = maxMedias
        self.selectable = selectable
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
    }
    
    public var body: some View {
        let medias = selectable ? pin.medias.sorted { !$0.isPrivate && $1.isPrivate } : pin.medias
        
        ScrollView(.horizontal) {
            HStack(spacing: 4) {
                ForEach(medias.prefix(maxMedias)) { media in
                    MediaThumbnailView(
                        mediaItem: media,
                        contentMode: .fit,
                        cornerRadius: 14,
                        hideBadges: selectable,
                        dismissBeforeMediaInfo: dismissBeforeMediaInfo
                    )
                    .frame(height: 96)
                    .opacity(selectable && media.isPrivate ? 0.5 : 1)
                    .overlay {
                        if selectable {
                            PrivateVideoMediaBadgesView(media: media).padding(4)
                        }
                    }
                }

                if medias.count > maxMedias {
                    VStack {
                        Text("+\(medias.count - maxMedias)")
                            .roundedFount(size: 24, weight: .semibold, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        Text("медиа")
                            .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    }.frame(76)
                }
            }.padding(.horizontal, 12)
        }.scrollIndicators(.hidden)
    }
}
