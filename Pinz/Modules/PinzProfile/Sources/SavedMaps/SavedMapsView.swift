import SwiftUI
import PinzUI

public struct SavedMapsView: View {

    @State private var viewModel: SavedMapsViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = SavedMapsViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Сохранённые карты")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
