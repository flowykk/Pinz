import SwiftUI
import MapKit
import PinzDomain
import PinzBase
import PinzUI

enum PinPlaceSectionIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case map = "map"
}

public struct PinPlaceSectionView: View {

    @Binding var pin: Pin
    @State private var position: MapCameraPosition
    @AppStorage("pinzMapStyle") private var mapStyleRawValue: String = PinzMapStyle.satelight.rawValue

    private static let defaultOffset: CLLocationDegrees = 0.00045
    private static let defaultZoom: CLLocationDegrees = 0.003

    public init(pin: Binding<Pin>) {
        self._pin = pin
        
        self._position = State(initialValue: {
            if let coordinate = pin.wrappedValue.coordinates {
                return .region(
                    MKCoordinateRegion(
                        center: CLLocationCoordinate2D(
                            latitude: coordinate.latitude + Self.defaultOffset,
                            longitude: coordinate.longitude
                        ),
                        span: MKCoordinateSpan(latitudeDelta: Self.defaultZoom, longitudeDelta: Self.defaultZoom)
                    )
                )
            }

            return .automatic
        }())
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle(PinzBaseStrings.PinInfo.Label.location)
                .padding(.leading, 12)

            if let coordinates = pin.coordinates {
                map(with: coordinates)
            } else {
                coordinatesAddingSetting
            }
        }
    }

    @ViewBuilder
    private func map(with coordinates: CLLocationCoordinate2D) -> some View {
        Map(position: $position) {
            Annotation(pin.name, coordinate: coordinates, anchor: .bottom) {
                PinAnnotationView(pin: pin)
            }
        }
        .onChange(of: pin.coordinates) { _, newValue in
            if let coordinate = newValue {
                position = .region(
                    MKCoordinateRegion(
                        center: CLLocationCoordinate2D(
                            latitude: coordinate.latitude + Self.defaultOffset,
                            longitude: coordinate.longitude
                        ),
                        span: MKCoordinateSpan(latitudeDelta: Self.defaultZoom, longitudeDelta: Self.defaultZoom)
                    )
                )
            } else {
                position = .automatic
            }
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

    private var coordinatesAddingSetting: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "tripPins",
                    leading: .iconTitle(PinPlaceSectionIcon.map, PinzBaseStrings.PinInfo.Button.addLocation),
                    trailing: .icon(PinPlaceSectionIcon.chevronRight)
                )),
            ],
        )
    }
}
