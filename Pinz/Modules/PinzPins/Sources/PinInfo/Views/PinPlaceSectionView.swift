import SwiftUI
import MapKit
import PinzDomain
import PinzUI

public struct PinPlaceSectionView: View {

    @State var mapStyle: PinzMapStyle = .scheme

    // Remove to Pin
    @State private var position: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 55.7558, longitude: 37.6173),
            span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
        )
    )

    public init() {

    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle("Местоположение")
                .padding(.leading, 12)

            Map(position: $position)
                .mapStyle(.hybrid)
                .clipShape(RoundedRectangle(cornerRadius: 26))
                .frame(height: 200)
                .allowsHitTesting(false)
        }
    }
}
