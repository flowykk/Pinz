import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class FeedViewModel {

    private let networkService: NetworkServiceProtocol
    enum Route {
        case back
        case openPost(Post)
    }

    enum Intent {
        case navigate(Route)
    }

    private let pageSize = 2 // TODO: change back to 20 after testing pagination
    private(set) var posts: [Post] = []
    private(set) var isLoading = false
    private(set) var hasReachedEnd = false
    private var currentOffset = 0
    var filters: FeedFilterModel = FeedFilterModel()

    private var router: AppRouting?

    init(networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case let .openPost(post):
                router?.navigateToPostInfo(post: post)
            }
        }
    }

    func applyFilters(_ newFilters: FeedFilterModel) async {
        filters = newFilters
        await fetchFeed()
    }

    func resetFilters() async {
        filters = FeedFilterModel()
        await fetchFeed()
    }

    func fetchFeed() async {
        currentOffset = 0
        hasReachedEnd = false
        posts = []
        await loadPage(replacing: true)
    }

    func loadMore() async {
        guard !isLoading, !hasReachedEnd else { return }
        await loadPage(replacing: false)
    }

    private func loadPage(replacing: Bool) async {
        isLoading = true
        defer { isLoading = false }
        do {
            let feed = try await networkService.getFeed(
                limit:    pageSize,
                offset:   currentOffset,
                category: filters.categoryParam,
                season:   filters.seasonParam,
                city:     filters.cityParam,
                country:  filters.countryParam,
                sortBy:   filters.sortByParam
            )

            let newPosts = feed.map { item -> Post in
                let trip = item.trip
                let tripMedia = item.media.enumerated().compactMap { index, media in
                    media.toMediaItem(id: index + 1)
                }
                let fallbackMedia = Array(item.pins.enumerated().flatMap { pinIndex, pin in
                        pin.mediaItems().enumerated().map { mediaIndex, media in
                            MediaItem(
                                id: pinIndex * 1000 + mediaIndex + 1,
                                isPrivate: media.isPrivate,
                                type: media.type,
                                mediaURL: media.mediaURL
                            )
                        }
                    })
                let postMedia = tripMedia.isEmpty ? fallbackMedia : tripMedia

                let mediasPerPin = Self.mediaBuckets(media: tripMedia, bucketCount: item.pins.count)

                let pins = item.pins.enumerated().map { index, pin in
                    let pinMedias = pin.mediaItems()
                    let fallbackMedias = index < mediasPerPin.count ? mediasPerPin[index] : []
                    return pin.toPin(
                        index: index,
                        medias: pinMedias.isEmpty ? fallbackMedias : pinMedias
                    )
                }

                return Post(
                    id: trip.id,
                    name: trip.name,
                    description: trip.description,
                    participants: trip.participantsCount ?? 0,
                    likes: trip.likesCount,
                    dislikes: trip.dislikesCount,
                    favorites: 0,
                    views: 0,
                    pins: pins,
                    media: postMedia
                )
            }

            if replacing {
                posts = newPosts
            } else {
                posts.append(contentsOf: newPosts)
            }
            currentOffset += feed.count
            hasReachedEnd = feed.count < pageSize
        } catch {
            print(error)
        }
    }

    private static func mediaBuckets(media: [MediaItem], bucketCount: Int) -> [[MediaItem]] {
        guard bucketCount > 0 else { return [] }
        var buckets = Array(repeating: [MediaItem](), count: bucketCount)

        for index in media.indices {
            buckets[index % bucketCount].append(media[index])
        }

        return buckets
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
