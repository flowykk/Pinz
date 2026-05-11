import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct PostInfoView: View {

    @State private var viewModel: PostFeedItemViewModel
    @Environment(\.appRouter) private var router

    public init(post: Post) {
        viewModel = PostFeedItemViewModel(post: post)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            VStack(spacing: 16) {
                tripMap
                    .padding(.horizontal, 12)
                statistics
                    .padding(.horizontal, 18)
                    .padding(.top, -4)
                DescriptionView(description: viewModel.post.description)
                    .padding(.horizontal, 12)
                DefaultPinsListView(
                    pins: viewModel.post.pins,
                    hideMediaBadges: true,
                    pinTapped: { _ in }
                )
                .padding(.bottom, 20)
            }
            .padding(.top, 8)
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { router?.pop() }
            )
        }, centerView: {
            HeaderTitle(viewModel.post.name)
        }, rightView: {
            StatisticView(
                icon: "person.2.fill",
                text: String(viewModel.post.participants),
                iconSize: 16,
                iconColor: PinzUIAsset.textSecondary
            )
        })
    }

    @ViewBuilder
    private var statistics: some View {
        let iconSize: CGFloat = 18
        if viewModel.post.isRecommended {
            EmptyView()
        } else {
            HStack(spacing: 12) {
                Button {
                    viewModel.dispatch(.like)
                } label: {
                    StatisticView(
                        icon: viewModel.post.isLiked ? "hand.thumbsup.fill" : "hand.thumbsup",
                        text: String(viewModel.post.likes),
                        iconSize: iconSize,
                        iconColor: PinzUIAsset.textPrimary
                    )
                }
                .buttonStyle(.plain)

                Button {
                    viewModel.dispatch(.dislike)
                } label: {
                    StatisticView(
                        icon: viewModel.post.isDisliked ? "hand.thumbsdown.fill" : "hand.thumbsdown",
                        text: String(viewModel.post.dislikes),
                        iconSize: iconSize,
                        iconColor: PinzUIAsset.textPrimary
                    )
                }
                .buttonStyle(.plain)

                Spacer()

                Button {
                    viewModel.dispatch(.toggleFavourite)
                } label: {
                    StatisticView(
                        icon: viewModel.post.isSaved ? "bookmark.fill" : "bookmark",
                        iconSize: iconSize,
                        iconColor: PinzUIAsset.textPrimary
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var tripMap: some View {
        TripMapView(position: $viewModel.position, pins: viewModel.post.pins)
            .aspectRatio(1, contentMode: .fit)
            .disabled(true)
            .overlay {
                VStack {
                    Spacer()
                    GradientView(style: .bottom, color: .black, height: 120)
                }
                .padding(.bottom, -85)
            }
            .clipShape(RoundedRectangle(cornerRadius: 26))
    }
}
