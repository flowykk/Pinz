import SwiftUI
import PinzUI
import PinzDomain

struct TripPinsView: View {
    @Environment(\.dismiss) var dismiss

    let pins: [Pin]

    init(pins: [Pin]) {
        self.pins = pins
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
            VStack(spacing: 8) {
                ForEach(pins.indices, id: \.self) { index in
                    PinView(pin: pins[index])
                    if index != pins.count - 1 {
                        Divider().padding(.leading, 12)
                    }
                }
            }.padding(.top, 60).padding(.bottom, 90)
        }.scrollIndicators(.hidden)
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
        ZStack {
            VStack {
                Spacer()

                LinearGradient(
                    gradient: Gradient(colors: [
                        PinzUIAsset.background.swiftUIColor,
                        Color.clear,
                    ]),
                    startPoint: .bottom,
                    endPoint: .top
                ).frame(height: 130)
            }.ignoresSafeArea()

            VStack {
                Spacer()
                buttons
            }.padding(12)
        }
    }

    private var buttons: some View {
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
