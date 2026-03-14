import SwiftUI
import PinzUI
import PinzDomain

struct TripsListPopupView: View {

    private let trips: [Trip]
    private let onTripTapped: (Trip) -> Void
    private let onDismiss: (() -> Void)?

    @Environment(\.dismiss) private var dismiss

    public init(
        trips: [Trip],
        onTripTapped: @escaping (Trip) -> Void,
        onDismiss: (() -> Void)? = nil
    ) {
        self.trips = trips
        self.onTripTapped = onTripTapped
        self.onDismiss = onDismiss
    }

    var body: some View {
        ZStack {
            ScrollView {
                DefaultTripsListView(trips: trips, onTripTapped: onTripTapped)
                    .padding(.top, 70).padding(.bottom, 90)
            }

            header
        }.background(PinzUIAsset.background.swiftUIColor)
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
                type: .slot(style: .primary, title: "Добавить путешествие"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor
            ) {}
        }
    }
}
