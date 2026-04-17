import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct SelectablePinsListView: View {

    @State private var viewModel: SelectablePinsListViewModel

    @Environment(\.appRouter) private var router

    public init(trip: Trip) {
        viewModel = SelectablePinsListViewModel(trip: trip)
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
                    HeaderTitle(PinzBaseStrings.SelectablePins.Title.main)
                })
            } content: {
                content
            }

            if viewModel.trip.pins.isEmpty {
                NoPinsPlaceholderView()
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    @ViewBuilder
    private var content: some View {
        let pins = viewModel.pins
        VStack(spacing: 8) {
            ForEach(pins.indices, id: \.self) { index in
                SelectablePinShortInfoView(
                    pin: pins[index],
                    hideTags: true,
                    isSelected: viewModel.isSelected(pins[index]),
                    onSelect: { viewModel.dispatch(.select(pins[index])) },
                    pinTapped: { pin in
                        viewModel.dispatch(.navigate(.pinInfo(pin)))
                    }
                )
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }.padding(.bottom, 24)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 6) {
                if viewModel.selectedPins.count > 1 {
                    PinzButton(
                        type: .slot(
                            style: .secondary(needBorder: true),
                            title: viewModel.allSelected ? PinzBaseStrings.SelectablePins.Button.deselectAll : PinzBaseStrings.SelectablePins.Button.selectAll
                        ),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.selectAll) }
                    )
                }

                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.done),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.postPreview)) }
                ).disabledWithOpacity(viewModel.selectedPins.isEmpty)
            }
        }
    }
}
