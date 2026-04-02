import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class SelectablePinsListViewModel {

    enum Route {
        case postPreview
        case pinInfo(Pin)
        case back
    }

    enum Intent {
        case navigate(Route)
        case select(Pin)
        case selectAll
    }

    let trip: Trip
    let pins: [Pin]
    var selectedPins: Set<String> = []
    
    var allSelected: Bool {
        let selectablePins = pins.filter { !$0.isPrivate }
        return !selectablePins.isEmpty && selectedPins.count == selectablePins.count
    }

    init(trip: Trip) {
        self.trip = trip
        self.pins = trip.pins.sorted { !$0.isPrivate && $1.isPrivate }
    }

    private let networkService = NetworkService()
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .postPreview:
                router?.navigateToPostPreview(
                    trip: trip,
                    selectedPins: pins.filter { selectedPins.contains($0.id) }
                )
            case let .pinInfo(pin):
                router?.navigateToPinInfo(pin: pin, updateAction: nil)
            case .back:
                router?.pop()
            }
        case let .select(pin):
            withAnimation(.easeInOut(duration: 0.3)) {
                if selectedPins.contains(pin.id) {
                    selectedPins.remove(pin.id)
                } else {
                    selectedPins.insert(pin.id)
                }
            }
        case .selectAll:
            withAnimation(.easeInOut(duration: 0.3)) {
                if allSelected {
                    selectedPins.removeAll()
                } else {
                    selectedPins = Set(pins.filter { !$0.isPrivate }.map { $0.id })
                }
            }
        }
    }
    
    func isSelected(_ pin: Pin) -> Bool {
        selectedPins.contains(pin.id)
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
