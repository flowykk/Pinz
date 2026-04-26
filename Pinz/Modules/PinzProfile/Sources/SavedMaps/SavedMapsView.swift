import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

public struct SavedMapsView: View {

    @State private var viewModel: SavedMapsViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = SavedMapsViewModel()
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
                    HeaderTitle(PinzBaseStrings.SavedMaps.Title.main)
                })
            } content: {
                if !viewModel.isLoading {
                    DefaultTripsListView(trips: viewModel.trips) { trip in
                        viewModel.dispatch(.selectTrip(trip))
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
            Task {
                try? await viewModel.asyncDispatch(.fetchFavouriteTrips)
            }
        }
    }
}
