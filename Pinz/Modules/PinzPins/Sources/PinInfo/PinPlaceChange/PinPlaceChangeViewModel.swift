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
    var currentCoordinate: CLLocationCoordinate2D?
    var position: MapCameraPosition
    var currentCamera: MapCamera?
    private let originalCoordinate: CLLocationCoordinate2D?
    private var router: AppRouting?
    private var onSave: ((CLLocationCoordinate2D?) -> Void)?
    
    var hasChanges: Bool {
        switch (currentCoordinate, originalCoordinate) {
        case let (current?, original?):
            let latDiff = abs(current.latitude - original.latitude)
            let lonDiff = abs(current.longitude - original.longitude)
            return latDiff > 0.0001 || lonDiff > 0.0001
        case (nil, nil):
            return false
        default:
            return true
        }
    }

    init(
        pin: Pin,
        onSave: @escaping (CLLocationCoordinate2D?) -> Void
    ) {
        self.pin = pin
        self.currentCoordinate = pin.coordinates
        self.originalCoordinate = pin.coordinates
        self.currentCamera = nil
        if let coordinate = pin.coordinates {
            self.position = .region(
                MKCoordinateRegion(
                    center: coordinate,
                    span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                )
            )
        } else {
            self.position = .automatic
        }
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
                    if let originalCoordinate {
                        position = .camera(
                            MapCamera(
                                centerCoordinate: originalCoordinate,
                                distance: camera.distance,
                                heading: camera.heading,
                                pitch: camera.pitch
                            )
                        )
                    } else {
                        position = .automatic
                    }
                }
            } else {
                withAnimation(.easeInOut(duration: 0.3)) {
                    if let originalCoordinate {
                        position = .region(
                            MKCoordinateRegion(
                                center: originalCoordinate,
                                span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                            )
                        )
                    } else {
                        position = .automatic
                    }
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
