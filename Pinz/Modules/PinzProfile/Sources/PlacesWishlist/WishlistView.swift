import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct WishlistView: View {

    @State private var viewModel: WishlistViewModel

    @Environment(\.appRouter) private var router

    private let isReadOnly: Bool

    public init(places: [DesiredPlace] = [], isReadOnly: Bool = false) {
        viewModel = WishlistViewModel(wishlist: places)
        self.isReadOnly = isReadOnly
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
        .task {
            if !isReadOnly {
                await viewModel.loadWishlist()
            }
        }
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.Wishlist.Title.main)
        }, rightView: {
            if !isReadOnly {
                PinzButton(
                    type: .icon(.plus),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.wishlistElementCreation)) }
                )
            }
        })
    }
}
