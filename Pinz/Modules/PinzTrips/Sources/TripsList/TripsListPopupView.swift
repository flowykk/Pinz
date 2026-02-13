import SwiftUI
import PinzUI
import PinzDomain

struct TripsListPopupView: View {

    private let trips: [Trip]
    private let onDismiss: (() -> Void)?

    @Environment(\.dismiss) private var dismiss

    public init(
        trips: [Trip],
        onDismiss: (() -> Void)? = nil
    ) {
        self.trips = trips
        self.onDismiss = onDismiss
    }

    var body: some View {
        ZStack {
            ScrollView {
                DefaultTripsListView(trips: trips)
                    .padding(.top, 60).padding(.bottom, 90)
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
