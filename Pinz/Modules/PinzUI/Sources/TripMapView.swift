import SwiftUI
import MapKit
import PinzDomain

public struct TripMapView: View {
    
    @Binding var position: MapCameraPosition
    @AppStorage("pinzMapStyle") private var mapStyleRawValue: String = PinzMapStyle.satelight.rawValue
    
    let pins: [Pin]
    let onPinTap: ((Pin) -> Void)?
    let polylineCoordinates: [CLLocationCoordinate2D]

    public init(
        position: Binding<MapCameraPosition>,
        pins: [Pin],
        polylineCoordinates: [CLLocationCoordinate2D] = [],
        onPinTap: ((Pin) -> Void)? = nil
    ) {
        self._position = position
        self.pins = pins
        self.polylineCoordinates = polylineCoordinates
        self.onPinTap = onPinTap
    }

    public var body: some View {
        Map(position: $position) {
            if polylineCoordinates.count >= 2 {
                MapPolyline(coordinates: polylineCoordinates)
                    .stroke(.white.opacity(0.85), style: StrokeStyle(lineWidth: 3, lineCap: .round, lineJoin: .round))
            }
            ForEach(pins) { pin in
                Annotation(pin.name, coordinate: pin.coordinates, anchor: .bottom) {
                    PinAnnotationView(pin: pin)
                        .if(onPinTap != nil) { view in
                            view.onTapGesture {
                                onPinTap?(pin)
                            }
                        }
                }
            }
        }
        .savedMapStyle(mapStyleRawValue)
        .mapControlVisibility(.hidden)
        .ignoresSafeArea()
        .toolbar(.hidden)
    }
}
