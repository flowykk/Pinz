import SwiftUI
import MapKit
import PinzDomain
import PinzUI

public struct PinPlaceSectionView: View {

    @Binding var pin: Pin
    @State var mapStyle: PinzMapStyle = .scheme
    @State private var position: MapCameraPosition

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
            SettingTitle("Местоположение")
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
            .mapStyle(.hybrid)
            .clipShape(RoundedRectangle(cornerRadius: 26))
            .frame(height: 200)
            .allowsHitTesting(false)

            SettingSubtitle("Нажмите, чтобы изменить местоположение пина")
                .padding(.top, -2)
                .padding(.leading, 12)
        }
    }
}
