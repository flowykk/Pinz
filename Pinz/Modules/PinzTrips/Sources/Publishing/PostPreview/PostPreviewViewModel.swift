import SwiftUI
import MapKit
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class PostPreviewViewModel {

    enum Route {
        case pinInfo(Pin)
        case back(by: Int = 1)
    }

    enum Intent {
        case navigate(Route)
    }

    let trip: Trip
    let selectedPins: [Pin]
    var position: MapCameraPosition

    init(
        trip: Trip,
        selectedPins: [Pin]
    ) {
        self.trip = trip
        self.selectedPins = selectedPins.map { pin in
            Pin(
                name: pin.name,
                description: pin.description,
                category: pin.category,
                medias: pin.medias.filter { !$0.isPrivate },
                isPrivate: pin.isPrivate,
                startDate: pin.startDate,
                endDate: pin.endDate,
                tags: pin.tags,
                coordinates: pin.coordinates
            )
        }
        self.position = self.selectedPins.calculateInitialMapPosition()
    }

    private let networkService = NetworkService()
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .pinInfo(pin):
                router?.navigateToPinInfo(pin: pin)
            case let .back(depth):
                router?.pop(by: depth)
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
