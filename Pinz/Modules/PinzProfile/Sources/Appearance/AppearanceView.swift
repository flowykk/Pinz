import SwiftUI
import PinzUI
import MapKit

struct AppearanceView: View {
    @Environment(\.dismiss) var dismiss

    @State private var viewModel = AppearanceViewModel()
    @State private var selectedIconIndex: Int? = 0
    @State private var position: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 55.7558, longitude: 37.6173),
            span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
        )
    )
    
    var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    dismiss()
                }
            }, centerView: {
                HeaderTitle("Оформление")
            })

            VStack(spacing: 12) {
                appIconSettings

                mapSettings

                Spacer()
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)
        }
    }

    private var appIconSettings: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle("Иконка приложения")
                .padding(.leading, 12)

            AppIconsGridView()
        }
    }

    private var mapSettings: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle("Вид карты")
                .roundedFount(size: 16, weight: .medium)
                .padding(.leading, 12)

            SegmentedPicker(
                selection: Binding(
                    get: { viewModel.state.mapStyle },
                    set: { viewModel.dispatch(.changeMapStyle($0)) }
                ),
                items: [.satelight, .scheme, .hybrid]
            )

            Map(position: $position)
                .mapStyle(viewModel.state.mapStyle.toMapKitMapStyle())
                .aspectRatio(1, contentMode: .fit)
                .clipShape(RoundedRectangle(cornerRadius: 16))
        }
    }
}

#Preview {
    AppearanceView()
}
