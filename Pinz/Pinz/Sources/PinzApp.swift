import SwiftUI
import PinzNavigation
import PinzTrips
import CoreLocation

import PinzBase
import PinzDomain
import PinzUI
import PinzPins

@main
struct PinzApp: App {
    @State private var router = AppRouter()

    var body: some Scene {
        WindowGroup {
            RootView(router: router) {
//                PinInfoView(pin: Pin.stubs()[0])

//                PinPlaceChangeView(pin: Pin.stubs()[0], onSave: { _ in })

//                PinStoryView(pins: Pin.stubs())
//                TripInfoView(
                TripView(trips: Trip.stubs())
            }.toolbar(.hidden)
        }
    }
}
