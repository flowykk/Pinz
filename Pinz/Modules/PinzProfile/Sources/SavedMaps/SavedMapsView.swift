import SwiftUI
import PinzUI

struct SavedMapsView: View {
    @Environment(\.dismiss) var dismiss

    var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    dismiss()
                }
            }, centerView: {
                HeaderTitle("Сохранённые карты")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }
}
