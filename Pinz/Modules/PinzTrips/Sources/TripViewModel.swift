import SwiftUI
import MapKit
import PinzDomain
import PinzBase

@Observable
public class TripViewModel {
    
    public enum Intent {
        case navigateToTripInfo
        case navigateToProfile(user: User)
        case navigateToFeed
        case navigateToMembers
        case navigateToPinInfo(pin: Pin)
        case selectPin(pin: Pin?)
        case unselectPin
    }
    
    var trip: Trip
    var position: MapCameraPosition
    var selectedPin: Pin?
    private var router: AppRouting?

    public init(trip: Trip) {
        self.trip = trip
        self.position = Self.calculateInitialPosition(for: trip)
    }
    
    private static func calculateInitialPosition(for trip: Trip) -> MapCameraPosition {
        guard !trip.pins.isEmpty else {
            return .automatic
        }
        
        let coordinates = trip.pins.map { $0.coordinates }
        
        let minLat = coordinates.map { $0.latitude }.min() ?? 0
        let maxLat = coordinates.map { $0.latitude }.max() ?? 0
        let minLon = coordinates.map { $0.longitude }.min() ?? 0
        let maxLon = coordinates.map { $0.longitude }.max() ?? 0
        
        let centerLat = (minLat + maxLat) / 2
        let centerLon = (minLon + maxLon) / 2
        
        let spanLat = (maxLat - minLat) * 1.5 // добавляем отступ
        let spanLon = (maxLon - minLon) * 1.5
        
        return .region(
            MKCoordinateRegion(
                center: CLLocationCoordinate2D(latitude: centerLat, longitude: centerLon),
                span: MKCoordinateSpan(
                    latitudeDelta: max(spanLat, 0.01),
                    longitudeDelta: max(spanLon, 0.01)
                )
            )
        )
    }
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case .navigateToTripInfo:
            router?.navigateToTripInfo(trip: trip)
        case let .navigateToProfile(user):
            router?.navigateToProfile(user: user)
        case .navigateToFeed:
            router?.navigateToFeed()
        case .navigateToMembers:
            // TODO: implement members navigation
            break
        case let .navigateToPinInfo(pin):
            router?.navigateToPinInfo(pin: pin)
        case let .selectPin(pin):
            selectedPin = pin
        case let .unselectPin:
            if let selectedPin {
                dispatch(.navigateToPinInfo(pin: selectedPin))
            }
            selectedPin = nil
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
