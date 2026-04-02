import SwiftUI
import PinzDomain
import PinzUI

struct ReviewPinView: View {

    let pin: Pin
    let index: Int
    let onTap: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            if !pin.medias.isEmpty {
                MediaGridView(
                    items: pin.medias.map {
                        MediaGridView.Item(
                            id: "\($0.id)",
                            url: $0.mediaURL?.absoluteString ?? "",
                            type: $0.type
                        )
                    }
                )
                .padding(.top, 6)
                .padding(.horizontal, 12)
            }
            if !pin.tags.isEmpty {
                TagsView(
                    tags: pin.tags,
                    onTagAdd: nil,
                    onTagDelete: nil,
                    style: .default
                )
                .padding(.top, 2)
                .padding(.horizontal, 12)
            }
        }
        .contentShape(Rectangle())
        .onTapGesture { onTap() }
    }

    private var header: some View {
        HeaderPinShortInfo(pin: pin, isPrivacyShown: false)
    }
}
