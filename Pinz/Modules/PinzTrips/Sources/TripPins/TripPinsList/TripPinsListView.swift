import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct TripPinsListView: View {

    @State private var viewModel: TripPinsListViewModel

    @Environment(\.appRouter) private var router

    public init(trip: Trip) {
        viewModel = TripPinsListViewModel(trip: trip)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                }, centerView: {
                    HeaderTitle(PinzBaseStrings.TripPins.title)
                })
            } content: {
                pinsList.padding(.bottom, 90)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    private var pinsList: some View {
        DefaultPinsListView(
            pins: viewModel.trip.pins,
            pinTapped: { pin in
                viewModel.dispatch(.navigate(.pinInfo(pin)))
            },
        )
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 6) {
                PinzButton(
                    type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addMedia),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { }
                )

                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.TripPins.Button.addPin),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.pinCreation)) }
                )
            }
        }
    }
}
