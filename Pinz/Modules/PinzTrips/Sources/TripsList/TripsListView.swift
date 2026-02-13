import SwiftUI
import PinzUI
import PinzDomain

public struct TripsListView: View {

    @State private var viewModel: TripsListViewModel

    @Environment(\.appRouter) private var router

    public init(trips: [Trip]) {
        viewModel = TripsListViewModel(trips: trips)
    }

    public var body: some View {
        CollapsibleHeader {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Путешествия")
            })
        } content: {
            DefaultTripsListView(trips: viewModel.trips)
                .padding(.bottom, 90)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
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
