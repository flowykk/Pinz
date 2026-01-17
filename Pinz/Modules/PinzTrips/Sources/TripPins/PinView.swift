import SwiftUI
import PinzUI
import PinzDomain

struct PinView: View {

    let pin: Pin

    init(pin: Pin) {
        self.pin = pin
    }

    var body: some View {
        VStack(spacing: 6) {
            header
            medias
            tags
        }
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

                Text(pin.category)
                    .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }

            Spacer()

            VStack {
                HStack {
                    StatisticView(text: String(pin.medias.count), icon: "photo.stack.fill")
                }
                Spacer()
            }
        }.padding(.horizontal, 16)
    }

    var medias: some View {
        ScrollView(.horizontal) {
            HStack(spacing: 4) {
                ForEach(pin.medias.prefix(6)) { media in
                    mediaView(media)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(96)
                        .cornerRadius(16)
                }

                if pin.medias.count > 6 {
                    Button {

                    } label: {
                        VStack {
                            Text("+\(pin.medias.count - 6)")
                                .roundedFount(size: 24, weight: .semibold, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                            Text("медиа")
                                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        }.frame(76)
                    }
                }
//                RoundedRectangle(cornerRadius: 16)
//                    .frame(96)
//                    .foregroundColor(.red)
            }.padding(.horizontal, 12)
        }.scrollIndicators(.hidden)
    }

    func mediaView(_ media: LoadedMedia) -> Image {
        switch media.content {
        case let .image(uIImage):
            Image(uiImage: uIImage)
        case let .video(url, firstFrame):
            Image(uiImage: firstFrame)
        }
    }

    @ViewBuilder
    var tags: some View {
        if let tags = pin.tags {
            TagsView(tags: tags, onTagAdd: {_ in }, onTagDelete: {_ in })
                .padding(.horizontal, 16)
        }
    }
}
