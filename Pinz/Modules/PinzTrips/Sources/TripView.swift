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
    @State private var isPinsListPresented = false
    @State private var isTripsListPresented = false
    @State private var isTripsListButtonRotated = false
    
    private let availableTrips: [Trip]

    @Environment(\.appRouter) private var router
    @Environment(\.dismiss) private var dismiss

    public init(trips: [Trip]) {
        self.availableTrips = trips

        let selectedTripID = SelectedTripStorage.shared.selectedTripID
        let trip = trips.first { $0.id == selectedTripID }
        viewModel = TripViewModel(trip: trip)
    }

    public var body: some View {
        ZStack {
            let hasSavedTrip = SelectedTripStorage.shared.selectedTripID != nil

            if viewModel.isLoading || (hasSavedTrip && viewModel.trip == nil) {
                LoadingView()
            } else if hasSavedTrip, let selectedTrip = viewModel.trip {
                TripMapView(
                    position: $viewModel.position,
                    pins: selectedTrip.pins,
                    onPinTap: { pin in
                        viewModel.dispatch(.selectPin(pin: pin))
                    }
                )

                gradient.ignoresSafeArea()
            } else {
                UnselectedTripView()
            }

            header

            if viewModel.state == .route {
                footer
            }
        }
        .onAppear {
            viewModel.setRouter(router)
            if SelectedTripStorage.shared.selectedTripID == nil {
                viewModel.dispatch(.clearSelection)
            }
            viewModel.dispatch(.checkAndUpdateTrip(availableTrips))
            Task {
                try? await viewModel.asyncDispatch(.loadSavedTrip)
                try? await viewModel.asyncDispatch(.loadCurrentProfile)
            }
//            TokenStorage.shared.save(
//                accessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Nzc1Mjk5OTksImlhdCI6MTc3NDkzNzk5OSwidXNlcl9pZCI6IjIxM2ExMjg3LTBkNDItNGM0ZS05YTlhLTBmNzhjMWRlZTg3MiIsInVzZXJuYW1lIjoidXNlciJ9.hhE9dSMOkata-Zw14hii9OBAybfwLapTOSzNIFobKMI",
//                refreshToken: "AVdEMNK7JAYPqE8qkWLVjJHnoVQcLu447hP3PCyilxQ="
//            )
        }
        .onChange(of: viewModel.trip?.id) { _, _ in
            Task { try? await viewModel.asyncDispatch(.loadSavedTrip) }
        }
        .onChange(of: isTripsListPresented, { _, newValue in
            withAnimation {
                isTripsListButtonRotated = newValue
            }
        })
        .ifLet(viewModel.trip) { view, selectedTrip in
            view.sheet(isPresented: $isPinsListPresented) {
                TripPinsListPopupView(pins: selectedTrip.pins) { pin in
                    isPinsListPresented = false
                    viewModel.dispatch(.navigate(.pinInfo(pin)))
                } createPinTapped: {
                    isPinsListPresented = false
                    viewModel.dispatch(.navigate(.pinCreation))
                }
                .pinzSheet()
                .presentationDetents([.medium, .large])
            }
        }
        .sheet(isPresented: $isTripsListPresented) {
            TripsListPopupView(selectedTripId: viewModel.trip?.id, onTripTapped: { selectedTrip in
                isTripsListPresented = false
                withAnimation(.easeInOut(duration: 0.3)) {
                    viewModel.dispatch(.selectTrip(selectedTrip))
                }
            }, onTripCreationTapped: {
                isTripsListPresented = false
                viewModel.dispatch(.navigate(.tripCreation))
            })
            .pinzSheet()
            .presentationDetents([.medium, .large])
        }
        .sheet(item: $viewModel.selectedPin) { pin in
            VStack(spacing: 8) {
                Spacer()

                DefaultPinShortInfoView(pin: pin, hideTags: true, dismissBeforeMediaInfo: true, pinTapped: { pin in
                    viewModel.dispatch(.unselectPin)
                })

                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.Trips.Button.viewDetails),
                    action: .plain { viewModel.dispatch(.unselectPin) }
                )
                .padding(.horizontal, 12)
            }
            .pinzSheet()
            .presentationDetents([.height(220)])
        }
    }

    @ViewBuilder
    private var header: some View {
        let hasSavedTrip = SelectedTripStorage.shared.selectedTripID != nil
        let selectedTrip = hasSavedTrip ? viewModel.trip : nil

        VStack {
            HStack(alignment: .top, spacing: 6) {
                VStack(alignment: .leading, spacing: 6) {
                    HStack(spacing: 6) {
                        if let selectedTrip {
                            tripHeader(for: selectedTrip)
                        }
                        button(.icon("chevron.up"), isRotated: isTripsListButtonRotated) {
                            isTripsListPresented = true
                        }
                    }
                    button(.icon("square.grid.2x2.fill")) {
                        viewModel.dispatch(.navigate(.feed))
                    }
                    if selectedTrip != nil {
                        button(
                            .icon(
                                viewModel.state == .default
                                ? "point.topright.arrow.triangle.backward.to.point.bottomleft.scurvepath.fill"
                                : "xmark"
                            ),
                            invertColors: true
                        ) {
                            viewModel.dispatch(.toggleRouteState)
                        }.disabledWithOpacity(selectedTrip?.pins.isEmpty == true)
                    }
                }

                Spacer()

                VStack(spacing: 6) {
                    button(.image(viewModel.currentUserAvatarImage ?? ImageProviderType.user.placeholder)) {
                        viewModel.dispatch(.navigate(
                            .profile(viewModel.currentUser ?? User(nickname: "user", email: ""))
                        ))
                    }
                    .disabledWithOpacity(viewModel.isProfileLoading || viewModel.isLoading)
                    if let selectedTrip {
                        button(.icon("list.bullet")) {
                            if !selectedTrip.pins.isEmpty {
                                isPinsListPresented = true
                            }
                        }
                        button(.icon("person.2.fill")) {
                            viewModel.dispatch(.navigate(.members))
                        }
                    }
                }
            }.padding(.horizontal, 10)

            Spacer()
        }
    }

    private func tripHeader(for trip: Trip) -> some View {
        Button {
            viewModel.dispatch(.navigate(.tripInfo))
        } label: {
            HStack(spacing: 8) {
                tripImage(for: trip)

                Text(trip.name)
                    .roundedFont(size: 16)
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

    @ViewBuilder
    private func tripImage(for trip: Trip) -> some View {
        if let localImage = trip.image {
            image(for: localImage)
        } else if let url = URL(string: trip.coverUrl ?? "") {
            LoadableImageThumbnail(url: url) { state in
                remoteTripImage(for: state)
            }
        } else {
            image(for: PinzDomainAsset.groupPlaceholder.image)
        }
    }

    @ViewBuilder
    private func remoteTripImage(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .frame(38)
                .cornerRadius(12)
                .overlay {
                    ProgressView()
                        .tint(.white)
                }
                .clipped()
        case .ready(let readyImage):
            image(for: readyImage)
        case .failure:
            image(for: PinzDomainAsset.groupPlaceholder.image)
        }
    }

    private func image(for uiImage: UIImage) -> some View {
        Image(uiImage: uiImage)
            .resizable()
            .scaledToFill()
            .frame(38)
            .cornerRadius(12)
            .clipped()
    }

    enum ButtonType {
        case icon(String)
        case image(UIImage)
    }

    @ViewBuilder
    private func button(
        _ type: ButtonType,
        invertColors: Bool = false,
        isRotated: Bool? = nil,
        action: @escaping () -> Void,
    ) -> some View {
        let foregroundColor = invertColors ? PinzUIAsset.backgroundSecondary.swiftUIColor : PinzUIAsset.textPrimary.swiftUIColor
        let backgroundColor = invertColors ? PinzUIAsset.textPrimary.swiftUIColor : PinzUIAsset.background.swiftUIColor
        let strokeColor = invertColors ? PinzUIAsset.textSecondary.swiftUIColor : PinzUIAsset.backgroundSecondary.swiftUIColor
        Button {
            action()
        } label: {
            RoundedRectangle(cornerRadius: Constants.buttonsCornerRadius)
                .strokeBorder(strokeColor, lineWidth: 2)
                .background(backgroundColor)
                .cornerRadius(Constants.buttonsCornerRadius)
                .frame(Constants.buttonsSize)
                .overlay {
                    Group {
                        switch type {
                        case let .icon(icon):
                            Image(systemName: icon)
                                .foregroundColor(foregroundColor)
                        case let .image(uIImage):
                            Image(uiImage: uIImage)
                                .resizable()
                                .scaledToFill()
                                .frame(38)
                                .cornerRadius(12)
                                .clipped()
                        }
                    }
                    .roundedFont(size: 20)
                    .ifLet(isRotated) { view, isRotated in
                        view.rotationEffect(.degrees(isRotated ? 0 : 180))
                    }
                }
        }.buttonStyle(.plain)
    }

    @ViewBuilder
    private var footer: some View {
        let pins = viewModel.sortedPins
        VStack {
            Spacer()
            HStack {
                button(.icon("chevron.left")) {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        viewModel.dispatch(.previousPin)
                    }
                }.disabledWithOpacity(viewModel.routePinIndex <= 0)

                Spacer()

                VStack(spacing: 2) {
                    Text(pins.isEmpty ? "" : pins[viewModel.routePinIndex].name)
                        .roundedFont(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .lineLimit(1)
                    Text("\(viewModel.routePinIndex + 1) / \(pins.count)")
                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                }

                Spacer()

                button(.icon("chevron.right")) {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        viewModel.dispatch(.nextPin)
                    }
                }.disabledWithOpacity(viewModel.routePinIndex >= pins.count - 1)
            }
            .padding(.horizontal, 10)
        }.padding(.bottom, 36)
    }

    @ViewBuilder
    private var gradient: some View {
        let gradientColor = PinzUIAsset.backgroundSecondary.swiftUIColor
        VStack {
            GradientView(style: .top, color: gradientColor, height: 150)
            Spacer()
            GradientView(
                style: .bottom,
                color: gradientColor,
                height: viewModel.state == .default ? 200 : 300
            ).padding(.bottom, -130)
        }
    }
}
