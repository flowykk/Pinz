import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct FeedView: View {

    @State private var viewModel: FeedViewModel
    @State private var isFilterPresented = false

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init() {
        viewModel = FeedViewModel()
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                LazyVStack(spacing: 24) {
                    ForEach(viewModel.posts) { post in
                        PostFeedItemView(
                            post: post,
                            recommendationFavouriteHandler: recommendationFavouriteHandler(for: post)
                        )
                        .transition(.move(edge: .bottom).combined(with: .opacity))
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
                    .animation(.spring(response: 0.35, dampingFraction: 0.9), value: viewModel.posts)
                    .padding(.top, -12)

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

                    if viewModel.shouldShowRecommendationButton {
                        Spacer(minLength: 110)
                    }
                }.padding(.vertical, 12)
            }

            if viewModel.shouldShowRecommendationButton {
                BottomGradientWithButtons {
                    PinzButton(
                        type: .slot(style: .primary, title: "Получить рекомендацию"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        disabled: viewModel.isRecommendationsLoading,
                        action: .async { await viewModel.loadRecommendation() }
                    )
                }
                .transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
        .animation(.easeInOut(duration: 0.25), value: viewModel.shouldShowRecommendationButton)
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
            Task {
                await viewModel.loadIfNeededOnAppear()
            }
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

    private func recommendationFavouriteHandler(
        for post: Post
    ) -> PostFeedItemViewModel.RecommendationFavouriteHandler? {
        guard post.isRecommended else { return nil }
        return { [weak viewModel] shouldSave in
            guard let viewModel else { throw CancellationError() }
            return try await viewModel.toggleRecommendationFavourite(shouldSave: shouldSave)
        }
    }
}
