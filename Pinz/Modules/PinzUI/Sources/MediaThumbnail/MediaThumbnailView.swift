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
                Image(uiImage: image)
                    .resizable()
                    .aspectRatio(square ? 1 : nil, contentMode: contentMode)
            case .failure:
                failureView
            }
        }
        .cornerRadius(cornerRadius)
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
