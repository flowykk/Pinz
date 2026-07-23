import SwiftUI

public struct SettingSubtitle: View {

    public enum Style {
        case `default`
        case destructive

        var color: Color {
            switch self {
            case .default: PinzUIAsset.textSecondary.swiftUIColor
            case .destructive: PinzUIAsset.accentRed.swiftUIColor
            }
        }
    }

    private let text: String
    private let style: Style

    public init(_ text: String, style: Style = .default) {
        self.text = text
        self.style = style
    }

    public var body: some View {
        Text(text)
            .roundedFont(size: 12, foregroundColor: style.color)
    }
}
