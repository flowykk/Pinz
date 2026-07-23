import SwiftUI
import PinzAccessibility

extension Setting.TextFieldSetting {
    public var view: some View {
        textField
            .roundedFont(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
            .applyPinzAccessibility(PinzElement.setting(for: id))
    }

    @ViewBuilder
    var textField: some View {
        switch style {
        case .default:
            TextField(placeholder, text: $text, axis: .horizontal)
                .frame(maxWidth: .infinity, minHeight: 52)
                .ifLet(focused) { view, value in view.focused(value) }
        case .multiline:
            TextField(placeholder, text: $text, axis: .vertical)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 16)
                .ifLet(focused) { view, value in view.focused(value) }
                .lineLimit(nil)
        }
    }
}
