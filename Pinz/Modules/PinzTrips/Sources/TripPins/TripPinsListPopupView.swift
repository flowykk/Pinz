import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

struct TripPinsListPopupView: View {
    @Environment(\.dismiss) var dismiss

    let pins: [Pin]
    let tripStatus: String?
    let pinTapped: (Pin) -> Void
    let createPinTapped: () -> Void
    let addMediaTapped: () -> Void
    let onMediaUpdated: ((MediaItem, Pin) -> Void)?

    init(
        pins: [Pin],
        tripStatus: String? = nil,
        pinTapped: @escaping (Pin) -> Void,
        createPinTapped: @escaping () -> Void,
        addMediaTapped: @escaping () -> Void = {},
        onMediaUpdated: ((MediaItem, Pin) -> Void)? = nil
    ) {
        self.pins = pins
        self.tripStatus = tripStatus
        self.pinTapped = pinTapped
        self.createPinTapped = createPinTapped
        self.addMediaTapped = addMediaTapped
        self.onMediaUpdated = onMediaUpdated
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
                    onMediaUpdated: onMediaUpdated
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

    private var addMediaStatusLabel: String? {
        switch tripStatus {
        case "ADD_MEDIA_UPLOADING": return PinzBaseStrings.TripPins.Status.uploading
        case "ADD_MEDIA_GROUPING_REVIEW": return PinzBaseStrings.TripPins.Status.groupingReview
        case "ADD_MEDIA_PROCESSING": return PinzBaseStrings.TripPins.Status.processing
        case "ADD_MEDIA_DRAFT_FINAL_REVIEW": return PinzBaseStrings.TripPins.Status.draftFinalReview
        default: return nil
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(alignment: .top, spacing: 6) {
                VStack(spacing: 4) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addMedia),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { addMediaTapped() }
                    )
                    addMediaStatusLabelView
                }

                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.TripPins.Button.addPin),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { createPinTapped() }
                )
            }.if(addMediaStatusLabel != nil) { view in
                view.padding(.bottom, -20)
            }
        }
    }

    @ViewBuilder
    private var addMediaStatusLabelView: some View {
        if let label = addMediaStatusLabel {
            Text(label)
                .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
        }
    }
}
