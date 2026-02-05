import SwiftUI
import PinzUI

public struct TripMembersView: View {

    @State private var viewModel: TripMembersViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = TripMembersViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle("Участники путешествия")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
