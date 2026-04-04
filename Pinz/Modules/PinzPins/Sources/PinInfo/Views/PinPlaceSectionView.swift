import SwiftUI
import MapKit
import PinzDomain
import PinzBase
import PinzUI

public struct PinPlaceSectionView: View {

    @Binding var pin: Pin
    @State private var position: MapCameraPosition
    @AppStorage("pinzMapStyle") private var mapStyleRawValue: String = PinzMapStyle.satelight.rawValue

    private let defaultOffset: CLLocationDegrees = 0.00045
    private let defaultZoom: CLLocationDegrees = 0.003

    public init(pin: Binding<Pin>) {
        self._pin = pin
        
        self._position = State(initialValue: .region(
            MKCoordinateRegion(
                center: CLLocationCoordinate2D(
                    latitude: pin.wrappedValue.coordinates.latitude + defaultOffset,
                    longitude: pin.wrappedValue.coordinates.longitude
                ),
                span: MKCoordinateSpan(latitudeDelta: defaultZoom, longitudeDelta: defaultZoom)
            )
        ))
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle(PinzBaseStrings.PinInfo.Label.location)
                .padding(.leading, 12)

            Map(position: $position) {
                Annotation(pin.name, coordinate: pin.coordinates, anchor: .bottom) {
                    PinAnnotationView(pin: pin)
                }
            }
            .onChange(of: pin.coordinates) { oldValue, newValue in
                position = .region(
                    MKCoordinateRegion(
                        center: CLLocationCoordinate2D(
                            latitude: newValue.latitude + defaultOffset,
                            longitude: newValue.longitude
                        ),
                        span: MKCoordinateSpan(latitudeDelta: defaultZoom, longitudeDelta: defaultZoom)
                    )
                )
            }
            .savedMapStyle(mapStyleRawValue)
            .mapControlVisibility(.hidden)
            .clipShape(RoundedRectangle(cornerRadius: 26))
            .frame(height: 200)
            .allowsHitTesting(false)

            SettingSubtitle(PinzBaseStrings.PinInfo.Hint.changeLocation)
                .padding(.top, -2)
                .padding(.leading, 12)
        }
    }
}
