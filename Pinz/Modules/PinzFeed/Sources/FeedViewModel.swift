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

    struct RecommendationState: Equatable {
        let snapshotToken: String
        let pinIds: [String]
        let city: String?
        let country: String?
        let category: String?
        let season: String?
        var savedTripId: String?
        var isSaving: Bool
    }

    private let pageSize = 2 // TODO: change back to 20 after testing pagination
    private(set) var posts: [Post] = []
    private(set) var isLoading = false
    private(set) var isRecommendationsLoading = false
    private(set) var shouldShowRecommendationButton = true
    private(set) var hasReachedEnd = false
    private(set) var hasLoadedFeed = false
    private(set) var recommendation: RecommendationState?
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
        Task { @MainActor in
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

    func loadRecommendation() async {
        guard !isRecommendationsLoading else { return }

        let (city, country) = normalizedLocation(from: filters)
        guard (city == nil) != (country == nil) else {
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.locationRequired)
            return
        }

        let category = filters.recommendationCategoryParam
        let season = filters.recommendationSeasonParam

        isRecommendationsLoading = true
        defer { isRecommendationsLoading = false }

        do {
            let response = try await networkService.getRecommendations(
                city: city,
                country: country,
                category: category,
                season: season
            )
            let map = response.map

            guard !map.pins.isEmpty, !map.snapshotToken.isEmpty else {
                recommendation = nil
                withAnimation(.spring(response: 0.35, dampingFraction: 0.85)) {
                    posts.removeAll(where: \.isRecommended)
                }
                showToast?(PinzBaseStrings.Feed.Recommendation.Toast.empty)
                return
            }

            recommendation = RecommendationState(
                snapshotToken: map.snapshotToken,
                pinIds: map.pins.map(\.id),
                city: city,
                country: country,
                category: category,
                season: season,
                savedTripId: nil,
                isSaving: false
            )

            let post = mapRecommendationToPost(map)
            withAnimation(.spring(response: 0.35, dampingFraction: 0.85)) {
                posts.removeAll(where: \.isRecommended)
                posts.insert(post, at: 0)
            }
            shouldShowRecommendationButton = false
        } catch let httpError as HTTPError {
            handleGetRecommendationsError(httpError)
        } catch {
            print(error)
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.loadFailed)
        }
    }

    private func resetRecommendationStateIfNeeded(for newFilters: FeedFilterModel) {
        let currentSignature = recommendationSignature(from: filters)
        let nextSignature = recommendationSignature(from: newFilters)

        guard currentSignature != nextSignature else { return }

        shouldShowRecommendationButton = true
        recommendation = nil
        posts.removeAll(where: \.isRecommended)
    }

    private func normalizedLocation(from filters: FeedFilterModel) -> (String?, String?) {
        (filters.cityParam, filters.countryParam)
    }

    private func recommendationSignature(
        from filters: FeedFilterModel
    ) -> (String?, String?, String?, String?) {
        let (city, country) = normalizedLocation(from: filters)
        return (city, country, filters.recommendationCategoryParam, filters.recommendationSeasonParam)
    }

    func toggleRecommendationFavourite(shouldSave: Bool) async throws -> String {
        guard var current = recommendation else {
            throw RecommendationFavouriteError.noActiveRecommendation
        }
        guard !current.isSaving else {
            throw RecommendationFavouriteError.alreadyInFlight
        }

        if shouldSave, current.savedTripId == nil {
            current.isSaving = true
            recommendation = current
            defer {
                if var state = recommendation {
                    state.isSaving = false
                    recommendation = state
                }
            }

            do {
                let response = try await networkService.saveRecommendation(
                    snapshotToken: current.snapshotToken,
                    pinIds: current.pinIds,
                    city: current.city,
                    country: current.country,
                    category: current.category,
                    season: current.season
                )
                let newId = response.trip.id
                if var state = recommendation {
                    state.savedTripId = newId
                    recommendation = state
                }
                if let idx = posts.firstIndex(where: { $0.isRecommended }) {
                    posts[idx].isSaved = true
                }
                return newId
            } catch let httpError as HTTPError {
                if httpError == .conflict {
                    showToast?(PinzBaseStrings.Feed.Recommendation.Toast.snapshotExpired)
                    await loadRecommendation()
                } else {
                    handleSaveRecommendationError(httpError)
                }
                throw httpError
            }
        }

        guard let tripId = current.savedTripId else {
            throw RecommendationFavouriteError.noSavedTripId
        }

        if shouldSave {
            _ = try await networkService.addTripToFavourites(id: tripId)
            if let idx = posts.firstIndex(where: { $0.isRecommended }) {
                posts[idx].isSaved = true
            }
        } else {
            try await networkService.removeTripFromFavourites(id: tripId)
            if let idx = posts.firstIndex(where: { $0.isRecommended }) {
                posts[idx].isSaved = false
            }
        }
        return tripId
    }

    private func handleGetRecommendationsError(_ error: HTTPError) {
        switch error {
        case .badRequest:
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.locationRequired)
        case .notFound:
            recommendation = nil
            withAnimation(.spring(response: 0.35, dampingFraction: 0.85)) {
                posts.removeAll(where: \.isRecommended)
            }
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.regionNotFound)
        case .unauthorized:
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.loadFailed)
        default:
            print("GET /recommendations error: \(error)")
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.loadFailed)
        }
    }

    private func handleSaveRecommendationError(_ error: HTTPError) {
        switch error {
        case .badRequest:
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.locationRequired)
        case .forbidden:
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.snapshotStale)
        case .notFound:
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.regionNotFound)
        default:
            print("POST /recommendations/save error: \(error)")
            showToast?(PinzBaseStrings.Feed.Recommendation.Toast.saveFailed)
        }
    }

    enum RecommendationFavouriteError: Error {
        case noActiveRecommendation
        case alreadyInFlight
        case noSavedTripId
    }

    private func mapRecommendationToPost(_ map: RecommendedMapDTO) -> Post {
        let trip = map.trip
        let domainTrip = trip.toTrip()
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
            ).censored(in: .public)
        }

        let publicTrip = domainTrip.censored(in: .public)
        return Post(
            id: trip.id,
            name: Self.recommendationDisplayTripName(from: publicTrip.name, map: map),
            description: Self.recommendedDescription(
                tripDescription: publicTrip.description,
                regionName: map.regionName,
                regionType: map.regionType
            ),
            category: domainTrip.category,
            season: domainTrip.season,
            participants: trip.participantsCount ?? 0,
            likes: trip.likesCount,
            dislikes: trip.dislikesCount,
            favorites: 0,
            views: 0,
            isRecommended: true,
            recommendedBadge: PinzBaseStrings.Feed.Recommendation.Badge.forYou,
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
        let domainTrip = trip.toTrip()
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
            ).censored(in: .public)
        }

        let publicTrip = domainTrip.censored(in: .public)
        return Post(
            id: trip.id,
            name: publicTrip.name,
            description: publicTrip.description,
            category: domainTrip.category,
            season: domainTrip.season,
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

    private static func recommendationDisplayTripName(from tripName: String, map: RecommendedMapDTO) -> String {
        guard let raw = map.regionName?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return tripName
        }
        let english = FeedGeoCatalog.englishDisplay(forRegionSlug: raw, regionType: map.regionType)
        if let range = tripName.range(of: raw, options: .caseInsensitive) {
            return tripName.replacingCharacters(in: range, with: english)
        }
        return tripName
    }

    private static func recommendedDescription(
        tripDescription: String?,
        regionName: String?,
        regionType: String?
    ) -> String {
        guard let rn = regionName?.trimmingCharacters(in: .whitespacesAndNewlines), !rn.isEmpty else {
            return tripDescription ?? ""
        }
        let pretty = FeedGeoCatalog.englishDisplay(forRegionSlug: rn, regionType: regionType)
        guard let tripDescription, !tripDescription.isEmpty else {
            return pretty
        }
        return "\(tripDescription) · \(pretty)"
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
