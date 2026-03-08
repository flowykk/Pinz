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

    let trip: Trip

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(trip: Trip) {
        self.trip = trip
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .pinInfo(pin):
                router?.navigateToPinInfo(pin: pin)
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
