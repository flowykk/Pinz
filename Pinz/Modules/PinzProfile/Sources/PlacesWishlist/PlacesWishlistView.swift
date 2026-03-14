import SwiftUI
import PinzUI

public struct PlacesWishlistView: View {

    @State private var viewModel: PlacesWishlistViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = PlacesWishlistViewModel()
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            VStack {
                let wishlist = viewModel.wishlist
                ForEach(wishlist.indices, id: \.self) { index in
                    WishlistElementView(element: wishlist[index]) { element in
                        // TODO: navigate to wishlist element screen
                    }.padding(.horizontal, 12)
                    if index != wishlist.count - 1 {
                        Divider().padding(.leading, 12)
                    }
                }
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                viewModel.dispatch(.navigate(.back))
            }
        }, centerView: {
            HeaderTitle("Желанные места")
        }, rightView: {
            PinzButton(type: .icon(.plus), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                // TODO: navigate to adding wishlist element form
            }
        })
    }
}
