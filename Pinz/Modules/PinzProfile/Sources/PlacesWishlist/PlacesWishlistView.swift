import SwiftUI
import PinzUI

public struct PlacesWishlistView: View {

    @State private var viewModel: PlacesWishlistViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = PlacesWishlistViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Желанные места")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
