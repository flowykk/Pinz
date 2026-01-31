import SwiftUI
import MapKit
import PinzDomain
import PinzUI

public struct PinPlaceSectionView: View {

    let pin: Pin
    @State var mapStyle: PinzMapStyle = .scheme
    @State private var position: MapCameraPosition

    public init(pin: Pin) {
        self.pin = pin
        
        self._position = State(initialValue: .region(
            MKCoordinateRegion(
                center: CLLocationCoordinate2D(
                    latitude: pin.coordinates.latitude + 0.00045,
                    longitude: pin.coordinates.longitude
                ),
                span: MKCoordinateSpan(latitudeDelta: 0.003, longitudeDelta: 0.003)
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
