import SwiftUI
import PinzBase
import PinzDomain

public struct MediaThumbnailView: View {

    private let image: UIImage
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat
    private let actionBeforeMediaInfo: (() -> Void)?

    @Environment(\.appRouter) private var router
    @Environment(\.dismiss) private var dismiss

    public init(
        image: UIImage,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
        actionBeforeMediaInfo: (() -> Void)? = nil
    ) {
        self.image = image
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.actionBeforeMediaInfo = actionBeforeMediaInfo
    }

    public var body: some View {
        imageView
            .contextMenu {
                Button {
                    actionBeforeMediaInfo?()
                    router?.navigateToMediaInfo(media: LoadedMedia(content: .image(PinzUIAsset.media1.image)))
                } label: {
                    Label("Детали", systemImage: "eye.fill")
                }

                Divider()
                Button(role: .destructive) {

                } label: {
                    Label("Удалить", systemImage: "trash")
                }
            } preview: {
                imageView
            }
    }

    private var imageView: some View {
        Image(uiImage: image)
            .resizable()
            .aspectRatio(contentMode: contentMode)
            .cornerRadius(cornerRadius)
            .overlay {
                VStack {
                    HStack {
                        let isPrivate = Bool.random()
                        badgeItem(
                            icon: isPrivate ? "lock.fill" : "lock.open.fill",
                        )
                        Spacer()
                    }
                    Spacer()
                }.padding(4)
            }
    }

    private func badgeItem(
        icon: String,
        color: PinzUIColors = PinzUIAsset.textPrimary,
    ) -> some View {
        RoundedRectangle(cornerRadius: 8)
            .fill(.ultraThinMaterial)
            .frame(24)
            .overlay {
                Image(systemName: icon)
                    .roundedFount(size: 12, foregroundColor: color.swiftUIColor)
            }
    }
}
