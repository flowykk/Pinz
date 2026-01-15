import SwiftUI

extension Setting.TextFieldSetting {
    var view: some View {
        textField
            .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
            .frame(maxWidth: .infinity, minHeight: 52)
    }

    @ViewBuilder
    var textField: some View {
        switch style {
        case .default:
            TextField(placeholder, text: $text, axis: .horizontal)
        case .multiline:
            TextField(placeholder, text: $text, axis: .vertical)
                .lineSpacing(6)
                .lineLimit(...15)
        }
    }
}
