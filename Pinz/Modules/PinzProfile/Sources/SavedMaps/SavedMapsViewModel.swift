import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class SavedMapsViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case selectTrip(Trip)
    }

    enum AsyncIntent {
        case fetchFavouriteTrips
    }

    var trips: [Trip] = []
    private(set) var isLoading = false

    private let networkService: NetworkServiceProtocol
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
            }
        case let .selectTrip(trip):
            router?.navigateToSavedTripDetail(trip: trip)
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .fetchFavouriteTrips:
            changeLoading(to: true)
            defer { changeLoading(to: false) }
            let dtos = try await networkService.getFavouriteTrips(
                limit: 100,
                offset: 0
            )
            trips = dtos.map { $0.toTrip() }
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }
}
