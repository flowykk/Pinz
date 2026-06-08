import SwiftUI
import PinzBase
import PinzDomain

public struct MediaThumbnailView: View {

    private let url: URL?
    private let type: MediaType
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let square: Bool
    private let cacheVariant: MediaCacheVariant
    private let cacheTargetPixel: Int

    public init(
        url: URL?,
        type: MediaType,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        square: Bool = false,
        cacheVariant: MediaCacheVariant = .thumbnail,
        cacheTargetPixel: Int = 560
    ) {
        self.url = url
        self.type = type
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.square = square
        self.cacheVariant = cacheVariant
        self.cacheTargetPixel = cacheTargetPixel
    }

    public var body: some View {
        switch type {
        case .image:
            LoadableImageThumbnail(
                url: url,
                cacheVariant: cacheVariant,
                cacheTargetPixel: cacheTargetPixel,
                content: contentBuilder
            )
        case .video:
            LoadableVideoThumbnail(url: url, content: contentBuilder)
        }
    }

    @ViewBuilder
    private func contentBuilder(_ state: LoadableMediaState) -> some View {
        Group {
            switch state {
            case .empty:
                loaderView
            case let .ready(image):
                if square {
                    squareImage(image)
                } else {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: contentMode)
                }
            case .failure:
                failureView
            }
        }
        .cornerRadius(cornerRadius)
    }

    @ViewBuilder
    private func squareImage(_ image: UIImage) -> some View {
        Color.clear
            .aspectRatio(1, contentMode: .fit)
            .overlay {
                Image(uiImage: image)
                    .resizable()
                    .modifier(ScaledImageContentMode(contentMode: contentMode))
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
            .clipped()
    }

    private var loaderView: some View {
        Rectangle()
            .fill(Color.gray.opacity(0.3))
            .aspectRatio(1, contentMode: .fit)
            .overlay {
                ProgressView()
                    .tint(.white)
            }
    }

    private var failureView: some View {
        Rectangle()
            .fill(Color.red.opacity(0.3))
            .aspectRatio(1, contentMode: .fit)
            .overlay {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundColor(.white)
            }
    }
}

private struct ScaledImageContentMode: ViewModifier {
    let contentMode: ContentMode

    func body(content: Content) -> some View {
        switch contentMode {
        case .fill:
            content.scaledToFill()
        case .fit:
            content.scaledToFit()
        @unknown default:
            content.scaledToFit()
        }
    }
}
