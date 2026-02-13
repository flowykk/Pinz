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
    }

    let trips: [Trip]

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(trips: [Trip], router: AppRouting? = nil) {
        self.trips = trips
        self.router = router
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
