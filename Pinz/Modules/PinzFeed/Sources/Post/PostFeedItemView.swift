import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

struct PostFeedItemView: View {

    @State private var viewModel: PostFeedItemViewModel
    @State private var selection: Int = 0

    init(
        post: Post,
        recommendationFavouriteHandler: PostFeedItemViewModel.RecommendationFavouriteHandler? = nil
    ) {
        viewModel = PostFeedItemViewModel(
            post: post,
            recommendationFavouriteHandler: recommendationFavouriteHandler
        )
    }

    var body: some View {
        let card = VStack(spacing: 0) {
            TabView(selection: $selection.animation()) {
                if hasMap {
                    map.tag(0)
                }
                mediaPages
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .frame(height: 300)
            .padding(.top, 4)

            TabViewProgressView(numberOfPages: max(totalTabPages, 1), currentIndex: selection)
                .padding(.top, 8)

            statistics
                .padding(.top, 6)
                .padding(.horizontal, 12)
        }
        .task {
            await viewModel.loadImages()
        }
        let cell = card
            .clipShape(RoundedRectangle(cornerRadius: 10))
        RecommendationPostCard(
            isRecommended: viewModel.post.isRecommended,
            badge: viewModel.post.recommendedBadge
        ) { cell }
    }

    @ViewBuilder
    public var statistics: some View {
        let iconSize: CGFloat = 18
        HStack(spacing: 12) {
            if !viewModel.post.isRecommended {
                Button {
                    viewModel.dispatch(.like)
                } label: {
                    StatisticView(
                        icon: viewModel.post.isLiked ? "hand.thumbsup.fill" : "hand.thumbsup",
                        text: String(viewModel.post.likes),
                        iconSize: iconSize,
                        iconColor: PinzUIAsset.textPrimary
                    )
                }.buttonStyle(.plain)

                Button {
                    viewModel.dispatch(.dislike)
                } label: {
                    StatisticView(
                        icon: viewModel.post.isDisliked ? "hand.thumbsdown.fill" : "hand.thumbsdown",
                        text: String(viewModel.post.dislikes),
                        iconSize: iconSize,
                        iconColor: PinzUIAsset.textPrimary
                    )
                }.buttonStyle(.plain)
            }

            Spacer()
            Button {
                viewModel.dispatch(.toggleFavourite)
            } label: {
                StatisticView(
                    icon: viewModel.post.isSaved ? "bookmark.fill" : "bookmark",
                    iconSize: iconSize,
                    iconColor: PinzUIAsset.textPrimary
                )
            }.buttonStyle(.plain)
        }
    }

    @ViewBuilder
    private var mediaPages: some View {
        if viewModel.post.media.isEmpty && !hasMap {
            placeholderPage.tag(0)
        } else {
            ForEach(viewModel.post.media.indices, id: \.self) { index in
                let media = viewModel.post.media[index]
                mediaPage(media: media, tagIndex: index + mediaOffset, mediaIndex: index)
            }
        }
    }

    private var hasMap: Bool {
        !viewModel.post.pins.isEmpty
    }

    private var tripTags: String? {
        let category = viewModel.post.category == .none ? nil : viewModel.post.category.value
        let season = viewModel.post.season == .none ? nil : viewModel.post.season.value
        let parts = [category, season].compactMap { $0 }
        return parts.isEmpty ? nil : parts.joined(separator: ", ")
    }

    private var totalTabPages: Int {
        viewModel.post.media.count + (hasMap ? 1 : 0)
    }

    private var mediaOffset: Int {
        hasMap ? 1 : 0
    }

    private var placeholderPage: some View {
        Rectangle()
            .fill(Color.gray.opacity(0.3))
            .frame(width: UIScreen.main.bounds.width, height: 300)
            .clipped()
            .cornerRadius(10)
    }

    private func mediaPage(media: MediaItem, tagIndex: Int, mediaIndex: Int) -> some View {
        Group {
            if media.type == .image, let image = viewModel.images[mediaIndex] {
                Image(uiImage: image)
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .clipped()
            } else {
                Rectangle()
                    .fill(Color.gray.opacity(0.3))
                    .overlay {
                        if media.type == .video {
                            Image(systemName: "play.fill")
                                .font(.system(size: 32, weight: .bold))
                                .foregroundColor(.white)
                        } else {
                            ProgressView().tint(.white)
                        }
                    }
            }
        }
        .frame(width: UIScreen.main.bounds.width, height: 300)
        .clipped()
        .cornerRadius(10)
        .tag(tagIndex)
    }

    var map: some View {
        TripMapView(
            position: $viewModel.position,
            pins: viewModel.post.pins
        )
        .frame(width: UIScreen.main.bounds.width, height: 300)
        .disabled(true)
        .overlay {
            VStack {
                GradientView(style: .top, color: PinzUIAsset.background.swiftUIColor, height: 150)
                Spacer()
            }.padding(.top, -60)
        }
        .overlay {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(viewModel.post.name)
                        .roundedFont(
                            size: 20,
                            weight: .bold,
                            foregroundColor: PinzUIAsset.background.swiftUIColor
                        )
                    if let tags = tripTags {
                        Text(tags)
                            .roundedFont(
                                size: 14,
                                weight: .semibold,
                                foregroundColor: PinzUIAsset.background.swiftUIColor
                            )
                    }
                    Spacer()
                }
                Spacer()
                StatisticView(
                    icon: "person.2.fill",
                    text: String(viewModel.post.participants),
                    iconSize: 16,
                    iconColor: PinzUIAsset.background
                )
            }
            .padding(.horizontal, 14)
            .padding(.top, 10)
        }.cornerRadius(10)
    }
}
