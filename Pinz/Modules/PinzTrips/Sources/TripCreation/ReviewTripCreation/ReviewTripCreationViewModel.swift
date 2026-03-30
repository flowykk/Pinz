import SwiftUI
import PinzBase

@Observable
final class ReviewTripCreationViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
