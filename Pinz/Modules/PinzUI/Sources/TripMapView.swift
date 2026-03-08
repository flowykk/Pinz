import SwiftUI
import MapKit
import PinzDomain

public struct TripMapView: View {
    
    @Binding var position: MapCameraPosition
    @AppStorage("pinzMapStyle") private var mapStyleRawValue: String = PinzMapStyle.satelight.rawValue
    
    let pins: [Pin]
    let onPinTap: ((Pin) -> Void)?

    public init(
        position: Binding<MapCameraPosition>,
        pins: [Pin],
        onPinTap: ((Pin) -> Void)? = nil
    ) {
        self._position = position
        self.pins = pins
        self.onPinTap = onPinTap
    }

    public var body: some View {
        Map(position: $position) {
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
