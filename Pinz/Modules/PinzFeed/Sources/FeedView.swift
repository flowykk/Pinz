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
            ScrollView {
                LazyVStack(spacing: 24) {
                    ForEach(viewModel.posts) { post in
                        PostView(post: post)
                            .contentShape(Rectangle())
                            .onTapGesture {
                                viewModel.dispatch(.navigate(.openPost(post)))
                            }
                    }
                }
                .padding(.vertical, 12)
            }
            .padding(.horizontal, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .task {
            await viewModel.fetchFeed()
        }
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
