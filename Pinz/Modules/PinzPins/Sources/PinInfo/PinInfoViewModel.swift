import SwiftUI
import PinzNetworking
import PinzDomain
import PinzUI
import PinzBase

@Observable
public class PinInfoViewModel {
    
    public enum State: SegmentedItem {
        public var id: Self { self }

        case info
        case gallery
        case editing

        public var content: SegmentedItemContent {
            switch self {
            case .info:
                .text("Информация")
            case .gallery:
                .text("Гелерея")
            case .editing:
                .text("")
            }
        }
    }

    public enum Intent {
        case changeState(State)

        case back
    }

    var state: State = .info

    var pin: Pin
    private let networkService = NetworkService()
    private var router: AppRouting?

    var isDefaultState: Bool {
        state == .info || state == .gallery
    }

    public init(pin: Pin) {
        self.pin = pin
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .changeState(futureState):
            changeState(to: futureState)
        case .back:
            router?.pop()
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
