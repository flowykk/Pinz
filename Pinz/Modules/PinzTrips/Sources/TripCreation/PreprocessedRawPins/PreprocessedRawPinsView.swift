import SwiftUI
import PinzUI
import PinzDomain
import PinzPins
import PinzBase

public struct PreprocessedRawPinsView: View {

    @State private var viewModel: PreprocessedRawPinsViewModel
    @State private var isMergePickerPresented = false

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init(tripId: String, pins: RawPins) {
        viewModel = PreprocessedRawPinsViewModel(tripId: tripId, pins: pins)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                if !viewModel.isLoading {
                    content
                }
            }

            if viewModel.isLoading {
                LoadingView()
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
        }
        .mergePinsSheet(isPresented: $isMergePickerPresented, pins: viewModel.pins.pins) { first, second in
            viewModel.dispatch(.mergePins(firstIndex: first, secondIndex: second))
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        })
    }

    private var content: some View {
        VStack {
            let pins = viewModel.pins.pins
            ForEach(pins.indices, id: \.self) { index in
                RawPinView(
                    pin: pins[index],
                    index: index,
                    allPins: pins,
                    onDeleteMedia: { media in
                        viewModel.dispatch(.deleteMedia(media, fromPin: pins[index].id))
                    },
                    onMoveMedia: { media, targetIndex in
                        viewModel.dispatch(.moveMedia(media, fromPin: index, toPin: targetIndex))
                    }
                )
                .padding(.horizontal, 12)
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }.padding(.bottom, 170)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons(height: 190) {
            VStack(spacing: 6) {
                HStack(spacing: 6) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.PreprocessedPins.Button.mergePins),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        disabled: viewModel.pins.pins.count < 2,
                        action: .plain { isMergePickerPresented = true }
                    )

                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addPin),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.addPin) }
                    )
                }
            }

            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: viewModel.isLoading,
                action: .async { try await viewModel.asyncDispatch(.continue) }
            )
            .accessibilityIdentifier("tripCreation.button.preprocessedNext")
        }
    }
}
