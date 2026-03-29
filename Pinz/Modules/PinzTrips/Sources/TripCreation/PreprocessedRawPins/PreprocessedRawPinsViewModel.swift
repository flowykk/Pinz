import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PreprocessedRawPinsViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    var pins: RawPins

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(pins: RawPins) {
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

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
