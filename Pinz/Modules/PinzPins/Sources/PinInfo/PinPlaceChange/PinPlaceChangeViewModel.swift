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
        case update(MapCameraUpdateContext)
    }

    var pin: Pin
    var currentCoordinate: CLLocationCoordinate2D
    var position: MapCameraPosition
    var currentCamera: MapCamera?
    private let originalCoordinate: CLLocationCoordinate2D
    private var router: AppRouting?
    private var onSave: ((CLLocationCoordinate2D) -> Void)?
    
    var hasChanges: Bool {
        let latDiff = abs(currentCoordinate.latitude - originalCoordinate.latitude)
        let lonDiff = abs(currentCoordinate.longitude - originalCoordinate.longitude)
        return latDiff > 0.0001 || lonDiff > 0.0001
    }

    init(
        pin: Pin,
        onSave: @escaping (CLLocationCoordinate2D) -> Void
    ) {
        self.pin = pin
        self.currentCoordinate = pin.coordinates
        self.originalCoordinate = pin.coordinates
        self.currentCamera = nil
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
            currentCoordinate = originalCoordinate
            if let camera = currentCamera {
                withAnimation(.easeInOut(duration: 0.3)) {
                    position = .camera(
                        MapCamera(
                            centerCoordinate: originalCoordinate,
                            distance: camera.distance,
                            heading: camera.heading,
                            pitch: camera.pitch
                        )
                    )
                }
            } else {
                withAnimation(.easeInOut(duration: 0.3)) {
                    position = .region(
                        MKCoordinateRegion(
                            center: originalCoordinate,
                            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                        )
                    )
                }
            }
        case let .update(context):
            currentCoordinate = context.region.center
            currentCamera = context.camera
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
