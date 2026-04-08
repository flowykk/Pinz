import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct FeedView: View {

    @State private var viewModel: FeedViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = FeedViewModel()
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            VStack {
                PostView(post: Post.stub)
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    public var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.Feed.Title.main)
        })
    }
}
