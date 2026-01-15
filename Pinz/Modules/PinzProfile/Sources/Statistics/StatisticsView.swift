import SwiftUI
import PinzUI

struct StatisticsView: View {
    @Environment(\.dismiss) var dismiss

    var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    dismiss()
                }
            }, centerView: {
                HeaderTitle("Статистика")
            })

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }
}
