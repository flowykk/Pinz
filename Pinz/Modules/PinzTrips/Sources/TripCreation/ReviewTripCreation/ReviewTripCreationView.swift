import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct ReviewTripCreationView: View {

    @State private var viewModel: ReviewTripCreationViewModel

    @Environment(\.appRouter) private var router

    public init(tripId: String, pins: [Pin]) {
        viewModel = ReviewTripCreationViewModel(tripId: tripId, pins: pins)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                VStack(spacing: 8) {
                    let pins = viewModel.pins
                    ForEach(pins.indices, id: \.self) { index in
                        ReviewPinView(pin: pins[index], index: index) {
                            viewModel.navigateToPinInfo(at: index, router: router)
                        }

                        if index != pins.count - 1 {
                            Divider().padding(.leading, 12)
                        }
                    }
                }
                .padding(.bottom, 100)
                .animation(.default, value: viewModel.pins)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.ReviewTripCreation.Title.main)
        }, rightView: {
            if viewModel.pinsHaveIssues {
                PinzButton(
                    type: .icon(.warning),
                    tint: PinzUIAsset.accentOrange.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.problems)) }
                )
            } else {
                PinzButton(
                    type: .icon(.checkmark),
                    tint: PinzUIAsset.accentGreen.swiftUIColor,
                    action: .plain {}
                )
            }
        })
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: viewModel.pinsHaveIssues,
                action: .async { try await viewModel.asyncDispatch(.finalize) }
            )
        }
    }
}
