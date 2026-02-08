import SwiftUI
import MapKit
import PinzDomain
import PinzBase

@Observable
public class TripViewModel {

    public enum Route {
        case tripInfo
        case profile(User)
        case feed
        case members
        case pinInfo(Pin)
    }

    public enum Intent {
        case navigate(Route)
        case selectPin(pin: Pin?)
        case unselectPin
    }
    
    var trip: Trip
    var position: MapCameraPosition
    var selectedPin: Pin?
    private var router: AppRouting?

    public init(trip: Trip) {
        self.trip = trip
        self.position = trip.pins.calculateInitialMapPosition()
    }
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .tripInfo:
                router?.navigateToTripInfo(trip: trip)
            case .profile(let user):
                router?.navigateToProfile(user: user)
            case .feed:
                router?.navigateToFeed()
            case .pinInfo(let pin):
                router?.navigateToPinInfo(pin: pin)
            case .members:
                router?.navigateToTripMembers()
            }
        case let .selectPin(pin):
            selectedPin = pin
        case .unselectPin:
            if let selectedPin {
                dispatch(.navigate(.pinInfo(selectedPin)))
            }
            selectedPin = nil
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
