import SwiftUI
import PinzUI

public struct ReviewTripCreationView: View {

    @State private var viewModel = ReviewTripCreationViewModel()

    @Environment(\.appRouter) private var router

    public init() {}

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
