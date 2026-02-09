import SwiftUI
import PinzUI
import PinzDomain

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
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        viewModel.dispatch(.navigate(.back))
                    }
                }, centerView: {
                    HeaderTitle("Пины путешествия")
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
