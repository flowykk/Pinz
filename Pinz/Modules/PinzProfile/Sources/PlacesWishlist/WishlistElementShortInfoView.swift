import SwiftUI
import PinzUI
import PinzDomain

struct WishlistElementShortInfoView: View {

    enum Constants {
        static let elementHeight: CGFloat = 125
        static let imageWidth: CGFloat = 100
    }

    let element: WishlistElement
    let onElementTap: (WishlistElement) -> Void

    var body: some View {
        Button {
            onElementTap(element)
        } label: {
            HStack(alignment: .center, spacing: 12) {
                Image(uiImage: element.image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: Constants.imageWidth, height: Constants.elementHeight)
                    .cornerRadius(16)

                VStack(alignment: .leading, spacing: 4) {
                    Text(element.title)
                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Text(element.description)
                        .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        .lineLimit(5)
                    Spacer(minLength: 0)
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
        }
        .buttonStyle(.plain)
        .frame(height: Constants.elementHeight)
    }
}
