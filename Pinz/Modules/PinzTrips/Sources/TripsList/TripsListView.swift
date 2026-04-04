import SwiftUI
import PinzUI
import PinzDomain

public struct TripsListView: View {

    @State private var viewModel: TripsListViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = TripsListViewModel()
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                }, centerView: {
                    HeaderTitle("Путешествия")
                })
            } content: {
                if !viewModel.isLoading {
                    DefaultTripsListView(trips: viewModel.trips) { trip in
                        withAnimation(.easeInOut(duration: 0.3)) {
                            viewModel.dispatch(.selectTrip(trip))
                        }
                    }
                    .padding(.bottom, 90)
                }
            }
            .background(PinzUIAsset.background.swiftUIColor)

            if viewModel.isLoading {
                LoadingView()
            }
        }
        .onAppear {
            viewModel.setRouter(router)
            Task { try await viewModel.asyncDispatch(.fetchTrips) }
        }
    }
}
