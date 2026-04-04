import SwiftUI
import PinzUI
import PinzBase

public struct FeedView: View {

    @State private var viewModel: FeedViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = FeedViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(PinzBaseStrings.Feed.Title.main)
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
