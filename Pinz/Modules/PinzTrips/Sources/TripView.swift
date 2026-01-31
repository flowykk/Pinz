import SwiftUI
import MapKit
import PinzUI
import PinzDomain
import PinzBase

public struct TripView: View {

    enum Constants {
        static let buttonsCornerRadius: CGFloat = 16
        static let buttonsSize: CGFloat = 50
    }

    @State private var viewModel: TripViewModel
    @State private var isPinsPresented = false
    @State private var position: MapCameraPosition = .automatic
    @Environment(\.appRouter) private var router

    public init(trip: Trip) {
        viewModel = TripViewModel(trip: trip)
        
        // Центрируем карту на первом пине, если есть
        if let firstPin = trip.pins.first {
            _position = State(initialValue: .region(
                MKCoordinateRegion(
                    center: firstPin.coordinates,
                    span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
                )
            ))
        }
    }

    public var body: some View {
        ZStack {
            Map(position: $position) {
                ForEach(viewModel.trip.pins) { pin in
                    Annotation(pin.name, coordinate: pin.coordinates, anchor: .bottom) {
                        PinAnnotationView(pin: pin)
                            .onTapGesture {
                                viewModel.dispatch(.navigateToPinInfo(pin: pin))
                            }
                    }
                }
            }
//                .mapStyle(.imagery(elevation: .realistic))
                .mapControlVisibility(.hidden)
                .ignoresSafeArea()
                .toolbar(.hidden)

            gradient.ignoresSafeArea()

            header
        }
        .onAppear { viewModel.setRouter(router) }
        .sheet(isPresented: $isPinsPresented) {
            TripPinsListView(pins: viewModel.trip.pins) { pin in
                isPinsPresented = false
                viewModel.dispatch(.navigateToPinInfo(pin: pin))
            }
            .pinzSheet()
            .presentationDetents([.medium, .large])
        }
    }

    private var header: some View {
        VStack {
            HStack(alignment: .top, spacing: 6) {
                VStack(alignment: .leading, spacing: 6) {
                    HStack(spacing: 6) {
                        tripHeader
                        button(.icon("chevron.down")) {

                        }
                    }
                    button(.icon("square.grid.2x2.fill")) {
                        viewModel.dispatch(.navigateToFeed)
                    }
                }

                Spacer()

                VStack(spacing: 6) {
                    button(.image(PinzUIAsset.avatar.image)) {
                        viewModel.dispatch(.navigateToProfile(user: User(nickname: "flowykk", email: "cristgames123@gmail.com")))
                    }
                    button(.icon("list.bullet")) {
                        if !viewModel.trip.pins.isEmpty {
                            isPinsPresented = true
                        }
                    }
                    button(.icon("person.2.fill")) {
                        viewModel.dispatch(.navigateToMembers)
                    }
                }
            }.padding(.horizontal, 10)

            Spacer()
        }
    }

    private var tripHeader: some View {
        Button {
            viewModel.dispatch(.navigateToTripInfo)
        } label: {
            HStack(spacing: 8) {
                Image(uiImage: viewModel.trip.image ?? PinzUIAsset.avatar.image)
                    .resizable()
                    .scaledToFill()
                    .frame(38)
                    .cornerRadius(12)
                    .clipped()

                Text(viewModel.trip.name)
                    .roundedFount(size: 16)
            }
            .padding(.leading, 6)
            .padding(.trailing, 10)
            .frame(height: Constants.buttonsSize)
            .background(
                RoundedRectangle(cornerRadius: Constants.buttonsCornerRadius)
                    .strokeBorder(PinzUIAsset.backgroundSecondary.swiftUIColor, lineWidth: 2)
                    .background(PinzUIAsset.background.swiftUIColor)
                    .cornerRadius(Constants.buttonsCornerRadius)
            )
        }.buttonStyle(.plain)
    }

    enum ButtonType {
        case icon(String)
        case image(UIImage)
    }

    @ViewBuilder
    private func button(
        _ type: ButtonType,
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
                    Group {
                        switch type {
                        case let .icon(icon):
                            Image(systemName: icon)
                        case let .image(uIImage):
                            Image(uiImage: uIImage)
                                .resizable()
                                .scaledToFill()
                                .frame(38)
                                .cornerRadius(12)
                                .clipped()
                        }
                    }
                    .roundedFount(size: 20)
                    .tint(PinzUIAsset.textPrimary.swiftUIColor)
                }
        }.buttonStyle(.plain)
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
}
