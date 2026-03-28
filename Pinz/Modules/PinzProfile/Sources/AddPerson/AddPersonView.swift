import SwiftUI
import PinzUI

struct AddPersonView: View {

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.xmark),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { dismiss() }
                )
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }
}
