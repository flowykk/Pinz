import SwiftUI
import PinzBase
import PinzDomain

public struct MediaThumbnailView: View {

    private let mediaItem: MediaItem
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let actionBeforeMediaInfo: (() -> Void)?

    @Environment(\.appRouter) private var router

    public init(
        mediaItem: MediaItem,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        actionBeforeMediaInfo: (() -> Void)? = nil
    ) {
        self.mediaItem = mediaItem
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.actionBeforeMediaInfo = actionBeforeMediaInfo
    }

    public var body: some View {
        Group {
            switch mediaItem.type {
            case .image:
                LoadableImageThumbnail(url: mediaItem.mediaURL, content: contentBuilder)
            case .video:
                LoadableVideoThumbnail(url: mediaItem.mediaURL, content: contentBuilder)
            }
        }.overlay {
            MediaBadgesView(media: mediaItem).padding(4)
        }
    }

    @ViewBuilder
    private func contentBuilder(_ state: LoadableMediaState) -> some View {
        Group {
            switch state {
            case .empty:
                loaderView
            case let .ready(image):
                let imageView = Image(uiImage: image)
                    .resizable()
                    .aspectRatio(contentMode: contentMode)
                imageView
                    .contextMenu {
                        Button {
                            actionBeforeMediaInfo?()
                            router?.navigateToMediaInfo(media: mediaItem)
                        } label: {
                            Label("Детали", systemImage: "eye.fill")
                        }

                        Divider()
                        Button(role: .destructive) {

                        } label: {
                            Label("Удалить", systemImage: "trash")
                        }
                    } preview: { imageView }
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
