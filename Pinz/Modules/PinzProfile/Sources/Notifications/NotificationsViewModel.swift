import SwiftUI
import PinzNetworking
import PinzBase

@Observable
final class NotificationsViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    private let networkService = NetworkService()
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

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
