import SwiftUI
import PinzUI

public struct StatisticsView: View {

    @State private var viewModel: StatisticsViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = StatisticsViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Статистика")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
