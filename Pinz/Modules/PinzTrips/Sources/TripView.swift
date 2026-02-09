import SwiftUI
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

    @Environment(\.appRouter) private var router
    @Environment(\.dismiss) private var dismiss

    public init(trip: Trip) {
        viewModel = TripViewModel(trip: trip)
    }

    public var body: some View {
        ZStack {
            TripMapView(
                position: $viewModel.position,
                pins: viewModel.trip.pins,
                onPinTap: { pin in
                    viewModel.dispatch(.selectPin(pin: pin))
                }
            )

            gradient.ignoresSafeArea()

            header
        }
        .onAppear { viewModel.setRouter(router) }
        .sheet(isPresented: $isPinsPresented) {
            TripPinsListPopupView(pins: viewModel.trip.pins) { pin in
                isPinsPresented = false
                viewModel.dispatch(.navigate(.pinInfo(pin)))
            }
            .pinzSheet()
            .presentationDetents([.medium, .large])
        }
        .sheet(item: $viewModel.selectedPin) { pin in
            VStack(spacing: 8) {
                Spacer()

                DefaultPinShortInfoView(pin: pin, hideTags: true, dismissBeforeMediaInfo: true, pinTapped: { pin in
                    viewModel.dispatch(.unselectPin)
                })

                PinzButton(type: .slot(style: .primary, title: "Посмотреть детали")) {
                    viewModel.dispatch(.unselectPin)
                }
                .padding(.horizontal, 12)
            }
            .pinzSheet()
            .presentationDetents([.height(220)])
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
                        viewModel.dispatch(.navigate(.feed))
                    }
                }

                Spacer()

                VStack(spacing: 6) {
                    button(.image(PinzUIAsset.avatar.image)) {
                        viewModel.dispatch(.navigate(.profile(User(nickname: "flowykk", email: "cristgames123@gmail.com"))))
                    }
                    button(.icon("list.bullet")) {
                        if !viewModel.trip.pins.isEmpty {
                            isPinsPresented = true
                        }
                    }
                    button(.icon("person.2.fill")) {
                        viewModel.dispatch(.navigate(.members))
                    }
                }
            }.padding(.horizontal, 10)

            Spacer()
        }
    }

    private var tripHeader: some View {
        Button {
            viewModel.dispatch(.navigate(.tripInfo))
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
            ).frame(height: 230)

            Spacer()
        }
    }
}
