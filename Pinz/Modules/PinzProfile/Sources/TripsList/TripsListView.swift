import SwiftUI
import PinzUI

public struct TripsListView: View {

    @State private var viewModel: TripsListViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = TripsListViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Путешествия")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
