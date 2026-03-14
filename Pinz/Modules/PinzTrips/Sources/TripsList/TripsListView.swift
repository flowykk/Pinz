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
            DefaultTripsListView(trips: viewModel.trips) { trip in
                withAnimation(.easeInOut(duration: 0.3)) {
                    viewModel.dispatch(.selectTrip(trip))
                }
            }
            .padding(.bottom, 90)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
