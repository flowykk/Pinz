import SwiftUI
import MapKit
import PinzNavigation
import PinzDomain

@Observable
public class TripViewModel {
    
    public enum Intent {
        case navigate(TripDestination)
    }

    var trip: Trip
    var navigator = Navigator<TripDestination>()

    public init(trip: Trip) {
        self.trip = trip
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(destination):
            navigator.navigate(to: destination)
        }
    }
}
