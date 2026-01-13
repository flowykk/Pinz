import Foundation
import SwiftUI

public enum Setting {

    public enum SettingAction {
        case plain(() -> Void)
        case async(() async throws -> Void)
    }

    public struct DefaultSetting {

        public enum Style {
            case `default`
            case destructive
        }

        public let id: String
        public let title: String
        public let value: String
        public let icon: String
        public let style: Style
        public let action: SettingAction

        public init(
            id: String = UUID().uuidString,
            title: String,
            value: String,
            icon: String,
            style: Style = .default,
            action: SettingAction
        ) {
            self.id = id
            self.title = title
            self.value = value
            self.icon = icon
            self.style = style
            self.action = action
        }

        var view: some View {
            HStack(spacing: 8) {
                Group {
                    Image(systemName: icon)
                        .font(.system(size: 18, weight: .medium))
                    Text(title)
                        .font(.system(size: 14, weight: .medium, design: .rounded))
                }
                .foregroundStyle(PinzUIAsset.textPrimary.swiftUIColor)

                Spacer()

                Group {
                    Text(value)
                    Image(systemName: "chevron.right")
                }
                .font(.system(size: 12, weight: .medium, design: .rounded))
                .foregroundStyle(PinzUIAsset.textSecondary.swiftUIColor)
            }
            .frame(height: 44)
        }
    }

    public struct TextFieldSetting {

        public enum Style {
            case `default`
            case multiline
        }

        public let id: String
        @Binding
        public var text: String
        public let placeholder: String
        public let style: Style

        public init(
            id: String = UUID().uuidString,
            text: Binding<String>,
            placeholder: String,
            style: Style = .default
        ) {
            self.id = id
            self._text = text
            self.placeholder = placeholder
            self.style = style
        }

        var view: some View {
            textField
                .font(.system(size: 14, weight: .medium, design: .rounded))
                .frame(maxWidth: .infinity, minHeight: 44)
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

    case `default`(DefaultSetting)
    case textField(TextFieldSetting)
}

public extension Setting {
    @ViewBuilder
    var view: some View {
        switch self {
        case let .default(setting):
            setting.view
        case let .textField(setting):
            setting.view
        }
    }
}

extension Setting: Identifiable {
    public var id: String {
        switch self {
        case let .default(setting):
            return setting.id
        case let .textField(setting):
            return setting.id
        }
    }
}
