import SwiftUI
import MapKit
import PinzDomain
import PinzBase

@Observable
final class TripViewModel {

    public enum Route {
        case tripInfo
        case profile(User)
        case feed
        case members
        case pinInfo(Pin)
        case pinCreation
    }

    enum State {
        case `default`
        case route

        mutating func toggle() {
            switch self {
            case .default: self = .route
            case .route: self = .default
            }
        }
    }

    enum Intent {
        case navigate(Route)
        case selectPin(pin: Pin?)
        case unselectPin
        case selectTrip(Trip)
        case checkAndUpdateTrip([Trip])

        case toggleRouteState
        case nextPin
        case previousPin
    }

    var state: State = .default
    var routePinIndex: Int = 0

    var trip: Trip?
    var _position: MapCameraPosition?
    var selectedPin: Pin?
    private var router: AppRouting?

    var sortedPins: [Pin] {
        (trip?.pins ?? []).sorted { ($0.startDate ?? .distantPast) < ($1.startDate ?? .distantPast) }
    }

    var position: MapCameraPosition {
        get {
            _position ?? .camera(MapCamera(
                centerCoordinate: CLLocationCoordinate2D(
                    latitude: 55.7558,
                    longitude: 37.6173
                ),
                distance: 50000
            ))
        }
        set { _position = newValue }
    }

    public init(trip: Trip?) {
        self.trip = trip
        if let trip {
            self._position = trip.pins.calculateInitialMapPosition()
        }
    }
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .tripInfo:
                if let trip {
                    router?.navigateToTripInfo(trip: trip)
                }
            case .profile(let user):
                router?.navigateToProfile(user: user)
            case .feed:
                router?.navigateToFeed()
            case .pinInfo(let pin):
                router?.navigateToPinInfo(pin: pin)
            case .pinCreation:
                router?.navigateToPinCreation()
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
            state = .default
            routePinIndex = 0
            SelectedTripStorage.shared.selectTrip(id: trip.id)
        case let .checkAndUpdateTrip(trips):
            guard let selectedTripID = SelectedTripStorage.shared.selectedTripID,
                  selectedTripID != trip?.id,
                  let newTrip = trips.first(where: { $0.id == selectedTripID }) else {
                return
            }
            dispatch(.selectTrip(newTrip))

        case .toggleRouteState:
            withAnimation(.easeInOut(duration: 0.3)) {
                state.toggle()
            }
            routePinIndex = 0
            if state == .route {
                navigateToRoutePin(at: 0)
            }
        case .nextPin:
            guard routePinIndex < sortedPins.count - 1 else { return }
            routePinIndex += 1
            navigateToRoutePin(at: routePinIndex)
        case .previousPin:
            guard routePinIndex > 0 else { return }
            routePinIndex -= 1
            navigateToRoutePin(at: routePinIndex)
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func navigateToRoutePin(at index: Int) {
        let pins = sortedPins
        guard !pins.isEmpty, index < pins.count else { return }
        withAnimation(.easeInOut(duration: 1)) {
            position = .camera(MapCamera(
                centerCoordinate: pins[index].coordinates,
                distance: 5000
            ))
        }
    }
}
