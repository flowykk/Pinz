import SwiftUI
import PinzUI
import PinzDomain

struct PinShortInfoView: View {

    let pin: Pin
    let hideTags: Bool
    let pinTapped: (Pin) -> Void

    @Environment(\.dismiss) private var dismiss

    init(
        pin: Pin,
        hideTags: Bool = false,
        pinTapped: @escaping (Pin) -> Void
    ) {
        self.pin = pin
        self.hideTags = hideTags
        self.pinTapped = pinTapped
    }

    var body: some View {
        Button {
            pinTapped(pin)
        } label: {
            VStack(spacing: 0) {
                header
                medias.padding(.top, 6)
                if !hideTags {
                    tags.padding(.top, 2)
                }
            }
        }.buttonStyle(.plain)
    }

    var header: some View {
        HStack {
            VStack(alignment: .leading) {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                        .roundedFount(size: 18)
                    Text(pin.name)
                        .roundedFount(size: 18)
                }

                Text(pin.category.value)
                    .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }

            Spacer()

            VStack {
                Spacer(minLength: 0)
                HStack(spacing: 10) {
                    StatisticView(icon: "photo.stack.fill", text: String(pin.medias.count))
                    StatisticView(
                        icon: pin.privacy == true ? "lock.fill" : "lock.open.fill",
                        iconColor: pin.privacy == true ? PinzUIAsset.accentRed : PinzUIAsset.accentGreen,
                    )
                }
                Spacer(minLength: 0)
            }
        }.padding(.horizontal, 16)
    }

    var medias: some View {
        ScrollView(.horizontal) {
            HStack(spacing: 4) {
                ForEach(pin.medias.prefix(6)) { media in
                    MediaThumbnailView(
                        image: mediaUIImage(media),
                        contentMode: .fit,
                        cornerRadius: 12,
                        actionBeforeMediaInfo: { dismiss() }
                    ).frame(height: 96)
                }

                if pin.medias.count > 6 {
                    VStack {
                        Text("+\(pin.medias.count - 6)")
                            .roundedFount(size: 24, weight: .semibold, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        Text("медиа")
                            .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    }.frame(76)
                }
            }.padding(.horizontal, 12)
        }.scrollIndicators(.hidden)
    }

    func mediaUIImage(_ media: LoadedMedia) -> UIImage {
        switch media.content {
        case let .image(uIImage):
            uIImage
        case let .video(url, firstFrame):
            firstFrame
        }
    }

    @ViewBuilder
    var tags: some View {
        if !pin.tags.isEmpty {
            TagsView(tags: pin.tags, onTagAdd: {_ in }, onTagDelete: {_ in }, style: .default)
                .padding(.horizontal, 16)
        }
    }
}
