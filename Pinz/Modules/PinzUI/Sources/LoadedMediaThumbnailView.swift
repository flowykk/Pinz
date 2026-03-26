import SwiftUI
import PinzDomain

public struct LoadedMediaThumbnailView: View {
    let media: LoadedMedia
    let contentMode: ContentMode
    let cornerRadius: CGFloat
    let onMediaDelete: () -> Void

    public init(
        media: LoadedMedia,
        contentMode: ContentMode = .fill,
        cornerRadius: CGFloat = 0,
        onMediaDelete: @escaping () -> Void
    ) {
        self.media = media
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.onMediaDelete = onMediaDelete
    }

    public var body: some View {
        Group {
            switch media.content {
            case .loading:
                loaderView
            case .image(let img):
                readyView(image: img)
            case .video(_, let frame):
                readyView(image: frame)
            }
        }.overlay {
            MediaBadgesView(trailingTopBadge: {
                HStack(spacing: 4) {
                    Button {
                        onMediaDelete()
                    } label: {
                        BadgeView(icon: .trash, color: PinzUIAsset.accentRed.swiftUIColor)
                    }

                    if case .video = media.content {
                        BadgeView(icon: .video)
                    }
                }
            }).padding(4)
        }
    }

    private func readyView(image: UIImage) -> some View {
        Image(uiImage: image)
            .resizable()
            .aspectRatio(contentMode: contentMode)
            .cornerRadius(cornerRadius)
    }

    private var loaderView: some View {
        Rectangle()
            .fill(Color.gray.opacity(0.3))
            .aspectRatio(1, contentMode: .fit)
            .cornerRadius(cornerRadius)
            .overlay {
                ProgressView()
                    .tint(.white)
            }
    }
}
