import SwiftUI

public struct MediaThumbnailView: View {

    private let image: UIImage
    private let contentMode: ContentMode
    private let cornerRadius: CGFloat

    public init(
        image: UIImage,
        contentMode: ContentMode,
        cornerRadius: CGFloat,
    ) {
        self.image = image
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
    }

    public var body: some View {
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
