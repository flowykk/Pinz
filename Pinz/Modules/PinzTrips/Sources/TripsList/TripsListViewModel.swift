import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class TripsListViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case selectTrip(Trip)
    }

    enum AsyncIntent {
        case fetchTrips
    }

    var trips: [Trip] = []
    private(set) var isLoading = false

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    init() {
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .selectTrip(trip):
            SelectedTripStorage.shared.selectTrip(id: trip.id)
            router?.pop(by: 2)
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .fetchTrips:
            changeLoading(to: true)
            let dtos = try await networkService.getTrips()
            let selectedTripID = SelectedTripStorage.shared.selectedTripID
            trips = dtos
                .map { $0.toTrip() }
                .filter { $0.id != selectedTripID }
            changeLoading(to: false)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }
}
