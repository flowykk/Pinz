import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct WishlistView: View {

    @State private var viewModel: WishlistViewModel

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    private let isReadOnly: Bool

    public init(places: [DesiredPlace] = [], isReadOnly: Bool = false) {
        viewModel = WishlistViewModel(wishlist: places)
        self.isReadOnly = isReadOnly
    }

    public var body: some View {
        ZStack {
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

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
        }
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
        })
    }

    @ViewBuilder
    private var gradientWithButtons: some View {
        if !isReadOnly {
            BottomGradientWithButtons {
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.Wishlist.Button.addNewPlace),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.wishlistElementCreation)) }
                )
            }
        }
    }
}
