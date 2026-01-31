import SwiftUI
import MapKit
import PinzBase
import PinzDomain

@Observable
final class PinPlaceChangeViewModel {

    enum Intent {
        case back
        case save
        case reset
    }

    var pin: Pin
    var currentCoordinate: CLLocationCoordinate2D
    var position: MapCameraPosition
    private let originalCoordinate: CLLocationCoordinate2D
    private var router: AppRouting?
    private var onSave: ((CLLocationCoordinate2D) -> Void)?
    
    var hasChanges: Bool {
        let latDiff = abs(currentCoordinate.latitude - originalCoordinate.latitude)
        let lonDiff = abs(currentCoordinate.longitude - originalCoordinate.longitude)
        return latDiff > 0.0001 || lonDiff > 0.0001
    }

    init(pin: Pin, onSave: @escaping (CLLocationCoordinate2D) -> Void) {
        self.pin = pin
        self.currentCoordinate = pin.coordinates
        self.originalCoordinate = pin.coordinates
        self.position = .region(
            MKCoordinateRegion(
                center: pin.coordinates,
                span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
            )
        )
        self.onSave = onSave
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .back:
            router?.pop()
        case .save:
            onSave?(currentCoordinate)
            router?.pop()
        case .reset:
            withAnimation(.easeInOut(duration: 0.3)) {
                currentCoordinate = originalCoordinate
                position = .region(
                    MKCoordinateRegion(
                        center: originalCoordinate,
                        span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                    )
                )
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
    
    func updateCoordinate(_ coordinate: CLLocationCoordinate2D) {
        currentCoordinate = coordinate
    }
}
