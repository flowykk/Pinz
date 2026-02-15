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
        case selectTrip(Trip)
        case checkAndUpdateTrip([Trip])
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
        case let .selectTrip(trip):
            self.trip = trip
            position = trip.pins.calculateInitialMapPosition()
            selectedPin = nil
            SelectedTripStorage.shared.selectTrip(id: trip.id)
        case let .checkAndUpdateTrip(trips):
            guard let selectedTripID = SelectedTripStorage.shared.selectedTripID,
                  selectedTripID != trip.id,
                  let newTrip = trips.first(where: { $0.id == selectedTripID }) else {
                return
            }
            dispatch(.selectTrip(newTrip))
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
