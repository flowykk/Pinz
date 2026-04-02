import SwiftUI
import PinzBase
import PinzDomain

@Observable
final class ReviewTripCreationViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    let tripId: String
    var pins: [Pin]

    private var router: AppRouting?

    init(tripId: String, pins: [Pin]) {
        self.tripId = tripId
        self.pins = pins
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

    func navigateToPinInfo(at index: Int, router: AppRouting?) {
        let pin = pins[index]
        router?.navigateToPinInfo(
            pin: pin,
            updateAction: PinUpdateAction { [weak self] updatedPin in
                self?.pins[index] = updatedPin
            }
        )
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
