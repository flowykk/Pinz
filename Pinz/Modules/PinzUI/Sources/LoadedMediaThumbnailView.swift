import SwiftUI
import PinzDomain

public struct LoadedMediaThumbnailView: View {
    let media: LoadedMedia
    let contentMode: ContentMode
    let cornerRadius: CGFloat

    public init(
        media: LoadedMedia,
        contentMode: ContentMode = .fill,
        cornerRadius: CGFloat = 0
    ) {
        self.media = media
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
    }

    public var body: some View {
        switch media.content {
        case .loading:
            loaderView
        case .image(let img):
            readyView(image: img)
        case .video(_, let frame):
            readyView(image: frame)
                .overlay {
                    MediaBadgesView(leadingTopBadge: {
                        BadgeView(icon: .video)
                    })
                    .padding(4)
                }
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
