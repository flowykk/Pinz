import SwiftUI

extension Setting.TextFieldSetting {
    public var view: some View {
        textField
            .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
            .frame(maxWidth: .infinity, minHeight: 52)
    }

    @ViewBuilder
    var textField: some View {
        switch style {
        case .default:
            TextField(placeholder, text: $text, axis: .horizontal)
                .ifLet(focused) { view, value in view.focused(value) }
        case .multiline:
            TextField(placeholder, text: $text, axis: .vertical)
                .ifLet(focused) { view, value in view.focused(value) }
                .lineSpacing(6)
                .lineLimit(...15)
        }
    }
}
