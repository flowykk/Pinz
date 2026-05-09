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
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            LazyVStack(spacing: 24) {
                if viewModel.shouldShowRecommendationButton {
                    SettingsGroup(settings: [recommendationSetting])
                        .padding(.horizontal, 12)
                }
                ForEach(viewModel.posts) { post in
                    PostFeedItemView(post: post)
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
            }.padding(.vertical, 12)
        }
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

    private var recommendationSetting: Setting {
        .default(Setting.DefaultSetting(
            id: "feed_recommendation",
            leading: .iconTitle(
                FeedRecommendationIcon.sparkles,
                "Рекомендация",
            ),
            trailing: viewModel.isRecommendationsLoading
                ? .values([.text("Загрузка...")])
                : .icon(FeedRecommendationIcon.chevronRight, PinzUIAsset.textSecondary.swiftUIColor),
            action: viewModel.isRecommendationsLoading ? nil : .plain {
                viewModel.requestRecommendationsButtonTapped()
            }
        ))
    }
}

private enum FeedRecommendationIcon: String, Setting.Icon {
    case sparkles = "wand.and.sparkles"
    case chevronRight = "chevron.right"
}
