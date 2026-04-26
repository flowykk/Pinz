import SwiftUI
import PinzUI
import PinzDomain

struct WishlistElementShortInfoView: View {

    enum Constants {
        static let elementHeight: CGFloat = 125
        static let imageWidth: CGFloat = 100
    }

    let element: DesiredPlace
    let onElementTap: (DesiredPlace) -> Void

    var body: some View {
        Button {
            onElementTap(element)
        } label: {
            HStack(alignment: .center, spacing: 12) {
                elementImage
                    .frame(width: Constants.imageWidth, height: Constants.elementHeight)
                    .cornerRadius(16)
                    .clipped()

                VStack(alignment: .leading, spacing: 4) {
                    Text(element.name)
                        .roundedFont(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Text(element.description)
                        .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        .lineLimit(5)
                    Spacer(minLength: 0)
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
        }
        .buttonStyle(.plain)
        .frame(height: Constants.elementHeight)
    }

    @ViewBuilder
    private var elementImage: some View {
        if let urlString = element.imageUrl, let url = URL(string: urlString) {
            LoadableImageThumbnail(url: url) { state in
                remoteImage(for: state)
            }
        } else {
            imagePlaceholder
        }
    }

    @ViewBuilder
    private func remoteImage(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            imagePlaceholder
        case .ready(let image):
            Image(uiImage: image)
                .resizable()
                .scaledToFill()
        case .failure:
            imagePlaceholder
        }
    }

    private var imagePlaceholder: some View {
        Rectangle()
            .fill(Color.gray.opacity(0.3))
    }
}
