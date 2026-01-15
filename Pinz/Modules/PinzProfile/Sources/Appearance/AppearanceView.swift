import SwiftUI
import PinzUI
import MapKit

struct AppearanceView: View {
    @Environment(\.dismiss) var dismiss

    @State private var viewModel = AppearanceViewModel()
    @State private var position: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 55.7558, longitude: 37.6173),
            span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
        )
    )
    
    var body: some View {
        VStack(spacing: 0) {
            PinzHeader(
                leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        dismiss()
                    }
                },
                centerView: {
//                    Text("Настройки оформления")
//                        .roundedFount(size: 18, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                }
            )

            VStack(alignment: .leading, spacing: 0) {
                appIconSettings

                mapSettings

                Spacer()
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)
        }
    }

    @ViewBuilder
    private var appIconSettings: some View {
        Text("Иконка приложения")
            .roundedFount(size: 18, weight: .medium)
            .padding(.leading, 12)

        ScrollView(.horizontal) {
            Group {
                Rectangle()
                Rectangle()
                Rectangle()
                Rectangle()
                Rectangle()
                Rectangle()
            }
            .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
            .frame(width: 120, height: 120)
        }
    }

    @ViewBuilder
    private var mapSettings: some View {
        Text("Вид карты")
            .roundedFount(size: 18, weight: .medium)
            .padding(.leading, 12)

        SegmentedPicker(selection: $viewModel.state.mapStyle, items: [.satelight, .scheme, .hybrid])
            .padding(.top, 6)

        Map(position: $position)
            .mapStyle(viewModel.state.mapStyle.toMapKitMapStyle())
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 16))
            .padding(.top, 6)
    }
}

#Preview {
    AppearanceView()
}
