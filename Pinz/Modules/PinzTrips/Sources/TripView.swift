import SwiftUI
import MapKit
import PinzUI
import PinzDomain

public struct TripView: View {

    public enum Constants {
        public static let buttonsCornerRadius: CGFloat = 16
        public static let buttonsSize: CGFloat = 50
    }

    @State private var viewModel: TripViewModel
    @State private var isPinsPresented = false
    @State private var position: MapCameraPosition = .automatic

    public init(trip: Trip) {
        viewModel = TripViewModel(trip: trip)
    }

    public var body: some View {
        NavigationStack(path: $viewModel.navigator.path) {
            ZStack {
                Map(position: $position)
                    .mapControlVisibility(.hidden)
                    .ignoresSafeArea()

                gradient.ignoresSafeArea()

                header.padding(.top, 8)
            }
            .navigationDestination(for: TripDestination.self) { destination in
                destinationView(for: destination).navigationBarHidden(true)
            }
            .navigationBarHidden(true)
            .sheet(isPresented: $isPinsPresented) {
                TripPinsView(pins: viewModel.trip.pins)
                    .presentationCornerRadius(40)
                    .presentationDetents([.medium, .large])
                    .presentationDragIndicator(.hidden)
            }
        }
    }

    private var header: some View {
        VStack {
            HStack(spacing: 6) {
                button(withIcon: "square.grid.2x2.fill") {
                    viewModel.dispatch(.navigate(.feed))
                }
                tripHeader
                button(withIcon: "person.2.fill") {
                    viewModel.dispatch(.navigate(.members))
                }
                button(withIcon: "list.bullet") {
                    if !viewModel.trip.pins.isEmpty {
                        isPinsPresented = true
                    }
                }
            }.padding(.horizontal, 10)

            Spacer()
        }
    }

    private var tripHeader: some View {
        RoundedRectangle(cornerRadius: Constants.buttonsCornerRadius)
            .strokeBorder(PinzUIAsset.backgroundSecondary.swiftUIColor, lineWidth: 2)
            .background(PinzUIAsset.background.swiftUIColor)
            .cornerRadius(Constants.buttonsCornerRadius)
            .frame(height: Constants.buttonsSize)
            .overlay {
                HStack {
                    Image(uiImage: viewModel.trip.image ?? PinzUIAsset.avatar.image)
                        .resizable()
                        .scaledToFill()
                        .frame(38)
                        .cornerRadius(12)
                        .clipped()
                        .padding(.leading, 6)

                    Text(viewModel.trip.name)
                        .roundedFount(size: 16)

                    Spacer(minLength: 0)
                }.frame(maxWidth: .infinity)
            }
    }

    @ViewBuilder
    private func button(
        withIcon icon: String,
        action: @escaping () -> Void
    ) -> some View {
        Button {
            action()
        } label: {
            RoundedRectangle(cornerRadius: Constants.buttonsCornerRadius)
                .strokeBorder(PinzUIAsset.backgroundSecondary.swiftUIColor, lineWidth: 2)
                .background(PinzUIAsset.background.swiftUIColor)
                .cornerRadius(Constants.buttonsCornerRadius)
                .frame(Constants.buttonsSize)
                .overlay {
                    Image(systemName: icon)
                        .roundedFount(size: 20)
                        .tint(PinzUIAsset.textPrimary.swiftUIColor)
                }
        }
    }

    private var gradient: some View {
        VStack {
            LinearGradient(
                gradient: Gradient(colors: [
                    Color.clear,
                    Color.black.opacity(0.8),
                ]),
                startPoint: .bottom,
                endPoint: .top
            ).frame(height: 170)

            Spacer()
        }
    }
    
    @ViewBuilder
    private func destinationView(for destination: TripDestination) -> some View {
        switch destination {
        case .members:
            TripMembersView()
        case .feed:
            Text("Feed")
        }
    }
}
