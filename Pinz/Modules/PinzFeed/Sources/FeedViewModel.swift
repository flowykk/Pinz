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
    private(set) var isRecommendationsLoading = false
    private(set) var shouldShowRecommendationButton = true
    private(set) var hasReachedEnd = false
    private(set) var hasLoadedFeed = false
    private var showToast: ((String) -> Void)?
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
        resetRecommendationStateIfNeeded(for: newFilters)
        filters = newFilters
        await fetchFeed()
    }

    func resetFilters() async {
        resetRecommendationStateIfNeeded(for: FeedFilterModel())
        filters = FeedFilterModel()
        await fetchFeed()
    }

    func fetchFeed() async {
        hasLoadedFeed = false
        currentOffset = 0
        hasReachedEnd = false
        posts = []
        await loadPage(replacing: true)
        hasLoadedFeed = true
    }

    func loadIfNeededOnAppear() async {
        guard !hasLoadedFeed, !isLoading else { return }
        await fetchFeed()
    }

    func loadMore() async {
        guard !isLoading, !hasReachedEnd else { return }
        await loadPage(replacing: false)
    }

    func requestRecommendationsButtonTapped() {
        guard !isRecommendationsLoading else {
            return
        }
        Task {
            await loadRecommendation()
        }
    }

    public func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
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

            let newPosts = feed.map(Self.mapToPost(item:))

            if replacing {
                withAnimation(.easeInOut(duration: 0.25)) {
                    posts = newPosts
                }
            } else {
                withAnimation(.easeInOut(duration: 0.25)) {
                    posts.append(contentsOf: newPosts)
                }
            }
            currentOffset += feed.count
            hasReachedEnd = feed.count < pageSize
        } catch {
            print(error)
        }
    }

    private func loadRecommendation() async {
        guard !isRecommendationsLoading else { return }

        let normalizedCity = filters.city.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedCountry = filters.country.trimmingCharacters(in: .whitespacesAndNewlines)
        let city = normalizedCity.isEmpty ? nil : normalizedCity
        let country = normalizedCountry.isEmpty ? nil : normalizedCountry
        guard (city == nil) != (country == nil) else {
            showToast?(L10n.recommendationLocationError)
            return
        }

        isRecommendationsLoading = true
        defer { isRecommendationsLoading = false }

        do {
            let response = try await networkService.getRecommendations(city: city, country: country)
            let recommendation = mapRecommendationToPost(response.map)
            withAnimation(.spring(response: 0.35, dampingFraction: 0.85)) {
                posts.removeAll(where: \.isRecommended)
                posts.insert(recommendation, at: 0)
            }
            shouldShowRecommendationButton = false
        } catch {
            print(error)
            showToast?(L10n.recommendationLoadFailed)
        }
    }

    private func resetRecommendationStateIfNeeded(for newFilters: FeedFilterModel) {
        let currentLocation = normalizedLocation(from: filters)
        let nextLocation = normalizedLocation(from: newFilters)

        guard currentLocation != nextLocation else { return }

        shouldShowRecommendationButton = true
        posts.removeAll(where: \.isRecommended)
    }

    private func normalizedLocation(from filters: FeedFilterModel) -> (String?, String?) {
        let normalizedCity = filters.city.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedCountry = filters.country.trimmingCharacters(in: .whitespacesAndNewlines)

        let city = normalizedCity.isEmpty ? nil : normalizedCity
        let country = normalizedCountry.isEmpty ? nil : normalizedCountry

        return (city, country)
    }

    private func mapRecommendationToPost(_ map: RecommendedMapDTO) -> Post {
        let trip = map.trip
        let tripMedia = (map.media ?? []).enumerated().compactMap { index, media in
            media.toMediaItem(id: index + 1)
        }
        let fallbackMedia = Array(map.pins.enumerated().flatMap { pinIndex, pin in
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
        let mediasPerPin = Self.mediaBuckets(media: tripMedia, bucketCount: map.pins.count)
        let pins = map.pins.enumerated().map { index, pin in
            let pinMedia = pin.mediaItems()
            let fallbackMedias = index < mediasPerPin.count ? mediasPerPin[index] : []
            return pin.toPin(
                index: index,
                medias: pinMedia.isEmpty ? fallbackMedias : pinMedia,
                fallbackTripId: trip.id,
                nameIfMissing: "Pin \(index + 1)"
            )
        }

        return Post(
            id: trip.id,
            name: trip.name,
            description: Self.recommendedDescription(
                tripDescription: trip.description,
                regionName: map.regionName,
                regionType: map.regionType
            ),
            participants: trip.participantsCount ?? 0,
            likes: trip.likesCount,
            dislikes: trip.dislikesCount,
            favorites: 0,
            views: 0,
            isRecommended: true,
            recommendedBadge: "Рекомендация для тебя",
            pins: pins,
            media: postMedia
        )
    }

    private static func mediaBuckets(media: [MediaItem], bucketCount: Int) -> [[MediaItem]] {
        guard bucketCount > 0 else { return [] }
        var buckets = Array(repeating: [MediaItem](), count: bucketCount)

        for index in media.indices {
            buckets[index % bucketCount].append(media[index])
        }

        return buckets
    }

    private static func mapToPost(item: FeedItemDTO) -> Post {
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
            isLiked: item.isLiked,
            isDisliked: item.isDisliked,
            isSaved: item.isSaved,
            pins: pins,
            media: postMedia
        )
    }

    private enum L10n {
        static let recommendationLocationError = "Выберите город или страну для рекомендаций."
        static let recommendationLoadFailed = "Не удалось загрузить рекомендацию."
    }

    private static func recommendedDescription(
        tripDescription: String?,
        regionName: String?,
        regionType: String?
    ) -> String {
        let region = [regionType, regionName].compactMap { $0 }.joined(separator: " ")
        guard !region.isEmpty else {
            return tripDescription ?? ""
        }
        guard let tripDescription, !tripDescription.isEmpty else {
            return region
        }
        return "\(tripDescription) · \(region)"
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
