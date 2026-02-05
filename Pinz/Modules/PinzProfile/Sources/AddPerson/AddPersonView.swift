import SwiftUI
import PinzUI

struct AddPersonView: View {

    @Environment(\.appRouter) private var router

    var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.xmark), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    router?.pop()
                }
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }
}
