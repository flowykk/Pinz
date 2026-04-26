import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class TripPinsListViewModel {

    enum Route {
        case pinInfo(Pin)
        case pinCreation
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    var trip: Trip

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    init(trip: Trip) {
        self.trip = trip
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .pinInfo(pin):
                router?.navigateToPinInfo(pin: pin, updateAction: PinUpdateAction { [weak self] updatedPin in
                    guard let self, let idx = trip.pins.firstIndex(where: { $0.serverId == updatedPin.serverId }) else { return }
                    trip.pins[idx] = updatedPin
                })
            case .pinCreation:
                router?.navigateToPinCreation()
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
