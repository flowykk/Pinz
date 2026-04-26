import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct FeedView: View {

    @State private var viewModel: FeedViewModel
    @State private var isFilterPresented = false

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = FeedViewModel()
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            LazyVStack(spacing: 24) {
                ForEach(viewModel.posts) { post in
                    PostFeedItemView(post: post)
                        .contentShape(Rectangle())
                        .onTapGesture {
                            viewModel.dispatch(.navigate(.openPost(post)))
                        }
                        .onAppear {
                            if post.id == viewModel.posts.last?.id {
                                Task { await viewModel.loadMore() }
                            }
                        }
                }
                if viewModel.isLoading && !viewModel.posts.isEmpty {
                    ProgressView()
                        .padding(.vertical, 16)
                        .frame(maxWidth: .infinity)
                } else if viewModel.hasReachedEnd {
                    Text("Больше постов нет")
                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 20)
                }
            }.padding(.vertical, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .task {
            await viewModel.fetchFeed()
        }
        .sheet(isPresented: $isFilterPresented) {
            FeedFilterView(
                currentFilters: viewModel.filters,
                onApply: { newFilters in
                    isFilterPresented = false
                    Task { await viewModel.applyFilters(newFilters) }
                },
                onReset: {
                    isFilterPresented = false
                    Task { await viewModel.resetFilters() }
                }
            )
            .pinzSheet()
            .presentationDetents([.medium, .large])
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
        }, rightView: {
            Button { isFilterPresented = true } label: {
                Image(systemName: "line.3.horizontal.decrease.circle")
                    .font(.system(size: 22))
                    .foregroundColor(
                        viewModel.filters.isActive
                            ? PinzUIAsset.accentGreen.swiftUIColor
                            : PinzUIAsset.textPrimary.swiftUIColor
                    )
            }
        })
    }
}
