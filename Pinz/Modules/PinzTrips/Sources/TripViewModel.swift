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
    }
    
    var trip: Trip
    private var router: AppRouting?

    public init(trip: Trip) {
        self.trip = trip
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
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
