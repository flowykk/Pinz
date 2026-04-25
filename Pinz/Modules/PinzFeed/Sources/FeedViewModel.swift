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

    private(set) var posts: [Post] = []

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

    func fetchFeed(
        limit: Int? = nil,
        offset: Int? = nil,
        category: String? = nil,
        season: String? = nil,
        locationId: Int? = nil,
        sortBy: String? = nil
    ) async {
        do {
            let feed = try await networkService.getFeed(
                limit: limit,
                offset: offset,
                category: category,
                season: season,
                locationId: locationId,
                sortBy: sortBy
            )

            posts = feed.map { trip in
                Post(
                    id: trip.id,
                    name: trip.name,
                    description: trip.description,
                    participants: trip.participantsCount ?? 0,
                    likes: trip.likesCount,
                    dislikes: trip.dislikesCount,
                    favorites: 0,
                    views: trip.mediaCount ?? 0,
                    pins: []
                )
            }
        } catch {
            print(error)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
