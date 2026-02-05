import SwiftUI
import PinzDomain
import PinzBase

@Observable
final class TripInfoViewModel {

    enum State {
        case `default`
        case editing
    }
    
    enum Intent {
        case changeState
        case setImage(UIImage?)
        case back
    }

    var state: State = .default

    var trip: Trip
    private var router: AppRouting?

    init(trip: Trip) {
        self.trip = trip
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .changeState:
            switch state {
            case .default: changeState(to: .editing)
            case .editing: changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                trip.image = newImage
            }
        case .back:
            router?.pop()
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
