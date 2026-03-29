import SwiftUI
import PinzUI
import MapKit

public struct AppearanceView: View {

    @State private var viewModel: AppearanceViewModel
    @State private var selectedIconIndex: Int? = 0
    @State private var position: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 55.7558, longitude: 37.6173),
            span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
        )
    )

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = AppearanceViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
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
        }.onAppear { viewModel.setRouter(router) }
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

            Group {
                switch viewModel.state.mapStyle {
                case .scheme:
                    Map(position: $position).mapStyle(.standard)
                case .satelight:
                    Map(position: $position).mapStyle(.imagery)
                case .hybrid:
                    Map(position: $position).mapStyle(.hybrid)
                }
            }
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 26))
            .disabled(true)
        }
    }
}

#Preview {
    AppearanceView()
}
