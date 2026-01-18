import SwiftUI
import MapKit
import PinzDomain
import PinzBase

@Observable
public class TripViewModel {
    
    public enum Intent {
        case navigateToProfile
        case navigateToFeed
        case navigateToMembers
    }
    
    var trip: Trip
    private var router: AppRouting?

    public init(trip: Trip) {
        self.trip = trip
    }
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case .navigateToProfile:
            router?.navigateToProfile()
        case .navigateToFeed:
            router?.navigateToFeed()
        case .navigateToMembers:
            // TODO: implement members navigation
            break
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
