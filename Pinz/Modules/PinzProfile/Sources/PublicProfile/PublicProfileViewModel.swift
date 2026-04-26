import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PublicProfileViewModel {

    enum Route {
        case back
        case wishlist
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case loadProfile
    }

    var isLoading: Bool = false
    var username: String = ""
    var avatarUrl: String?
    var desiredPlaces: [DesiredPlaceDTO] = []
    private var hasLoaded = false

    private let userId: String
    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?

    init(userId: String, networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.userId = userId
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case .wishlist:
                router?.navigateToPublicWishlist(places: desiredPlaces.map { $0.toDesiredPlace() })
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .loadProfile:
            guard !hasLoaded else { return }
            isLoading = true
            defer { isLoading = false }
            let response = try await networkService.getPublicUserProfile(id: userId)
            username = response.username
            avatarUrl = response.avatarUrl
            desiredPlaces = response.desiredPlaces
            hasLoaded = true
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
