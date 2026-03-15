import SwiftUI
import PinzUI

public struct WishlistView: View {

    @State private var viewModel: WishlistViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = WishlistViewModel()
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            VStack {
                let wishlist = viewModel.wishlist
                ForEach(wishlist.indices, id: \.self) { index in
                    WishlistElementShortInfoView(element: wishlist[index]) { element in
                        viewModel.dispatch(.navigate(.wishlistElement(element)))
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
                viewModel.dispatch(.navigate(.wishlistElementCreation))
            }
        })
    }
}
