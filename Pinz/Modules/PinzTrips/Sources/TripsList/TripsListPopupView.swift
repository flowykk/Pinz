import SwiftUI
import PinzUI
import PinzDomain

struct TripsListPopupView: View {

    @State private var viewModel: TripsListPopupViewModel

    private let selectedTripId: String?
    private let onTripTapped: (Trip) -> Void
    private let onTripCreationTapped: () -> Void
    private let onDismiss: (() -> Void)?

    @Environment(\.dismiss) private var dismiss

    public init(
        selectedTripId: String? = nil,
        onTripTapped: @escaping (Trip) -> Void,
        onTripCreationTapped: @escaping () -> Void,
        onDismiss: (() -> Void)? = nil
    ) {
        viewModel = TripsListPopupViewModel()
        self.selectedTripId = selectedTripId
        self.onTripTapped = onTripTapped
        self.onTripCreationTapped = onTripCreationTapped
        self.onDismiss = onDismiss
    }

    var body: some View {
        ZStack {
            ScrollView {
                if !viewModel.isLoading {
                    DefaultTripsListView(trips: viewModel.trips, onTripTapped: onTripTapped)
                        .padding(.top, 70).padding(.bottom, 90)
                }
            }

            header

            gradientWithButtons

            if viewModel.isLoading {
                LoadingView()
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            Task {
                try await viewModel.asyncDispatch(.fetchTrips(selectedTripId: selectedTripId ?? ""))
            }
        }
    }

    @ViewBuilder
    private var header: some View {
        VStack {
            GradientView(style: .top, color: PinzUIAsset.background.swiftUIColor, opacity: 1.0, height: 70)
            Spacer()
        }

        VStack {
            Text("Твои путешествия")
                .roundedFount(size: 20, weight: .semibold)
                .padding(.top, 16)
            Spacer()
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: "Создать путешествия"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                action: .plain { onTripCreationTapped() }
            )
        }
    }
}
