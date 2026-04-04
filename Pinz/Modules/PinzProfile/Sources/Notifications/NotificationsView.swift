import SwiftUI
import PinzUI
import PinzBase

public struct NotificationsView: View {

    @State private var viewModel: NotificationsViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = NotificationsViewModel()
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(PinzBaseStrings.Notifications.Title.main)
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }
}
