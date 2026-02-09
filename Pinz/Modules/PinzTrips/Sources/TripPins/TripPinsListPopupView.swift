import SwiftUI
import PinzUI
import PinzDomain

struct TripPinsListPopupView: View {
    @Environment(\.dismiss) var dismiss

    let pins: [Pin]
    let pinTapped: (Pin) -> Void

    init(pins: [Pin], pinTapped: @escaping (Pin) -> Void) {
        self.pins = pins
        self.pinTapped = pinTapped
    }

    var body: some View {
        ZStack {
            pinsView

            header

            gradientWithButtons
        }.background(PinzUIAsset.background.swiftUIColor)
    }

    private var pinsView: some View {
        ScrollView {
            DefaultPinsListView(
                pins: pins,
                dismissBeforeMediaInfo: true,
                pinTapped: pinTapped,
            ).padding(.top, 60).padding(.bottom, 90)
        }
        .scrollIndicators(.hidden)
        .animationsDisabled()
    }

    @ViewBuilder
    private var header: some View {
        VStack {
            LinearGradient(
                gradient: Gradient(colors: [
                    Color.clear,
                    PinzUIAsset.background.swiftUIColor.opacity(0.8),
                    PinzUIAsset.background.swiftUIColor,
                ]),
                startPoint: .bottom,
                endPoint: .top
            ).frame(height: 70)
            Spacer()
        }

        VStack {
            Text("Пины путешествия")
                .roundedFount(size: 20, weight: .semibold)
                .padding(.top, 16)
            Spacer()
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 6) {
                PinzButton(
                    type: .slot(style: .secondary(needBorder: true), title: "Добавить медиа"),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor
                ) {}

                PinzButton(
                    type: .slot(style: .primary, title: "Добавить пин"),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor
                ) {}
            }
        }
    }
}
