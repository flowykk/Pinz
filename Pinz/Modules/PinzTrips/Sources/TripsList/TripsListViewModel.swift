import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class TripsListViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case selectTrip(Trip)
    }

    var trips: [Trip]

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(trips: [Trip], router: AppRouting? = nil) {
        // Фильтруем уже выбранное путешествие
        let selectedTripID = SelectedTripStorage.shared.selectedTripID
        self.trips = trips.filter { $0.id != selectedTripID }
        self.router = router
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

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
