import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

struct TripPinsListPopupView: View {
    @Environment(\.dismiss) var dismiss

    let pins: [Pin]
    let pinTapped: (Pin) -> Void
    let createPinTapped: () -> Void
    let addMediaTapped: () -> Void

    init(
        pins: [Pin],
        pinTapped: @escaping (Pin) -> Void,
        createPinTapped: @escaping () -> Void,
        addMediaTapped: @escaping () -> Void
    ) {
        self.pins = pins
        self.pinTapped = pinTapped
        self.createPinTapped = createPinTapped
        self.addMediaTapped = addMediaTapped
    }

    var body: some View {
        ZStack {
            pinsView

            header

            gradientWithButtons
        }.background(PinzUIAsset.background.swiftUIColor)
    }

    @ViewBuilder
    private var pinsView: some View {
        if pins.isEmpty {
            Spacer()
            NoPinsPlaceholderView()
            Spacer()
        } else {
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
    }

    @ViewBuilder
    private var header: some View {
        VStack {
            GradientView(style: .top, color: PinzUIAsset.background.swiftUIColor, opacity: 1.0, height: 50)
            Spacer()
        }

        VStack {
            Text(PinzBaseStrings.TripPins.title)
                .roundedFont(size: 20, weight: .semibold)
                .padding(.top, 16)
            Spacer()
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 6) {
                PinzButton(
                    type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addMedia),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { addMediaTapped() }
                )

                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.TripPins.Button.addPin),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { createPinTapped() }
                )
            }
        }
    }
}
