import Foundation
import SwiftUI

public enum Setting {

    public protocol Icon {
        var rawValue: String { get }
    }

    public enum SettingAction {
        case plain(() -> Void)
        case async(() async throws -> Void)
    }

    public struct DefaultSetting {

        public enum Style {
            case `default`
            case destructive
        }

        public enum Value: Identifiable {
            case text(String)
            case icon(Icon, Color)

            public var id: String {
                switch self {
                case let .text(str): return "string_\(str)"
                case let .icon(icon, _): return "icon_\(icon.rawValue)"
                }
            }
        }

        let id: String
        let title: String
        let values: [Value]
        let icon: Icon
        let trailIcon: Icon?
        let style: Style
        let action: SettingAction

        var titleColor: Color {
            switch style {
            case .default: PinzUIAsset.textPrimary.swiftUIColor
            case .destructive: PinzUIAsset.accentRed.swiftUIColor
            }
        }

        var trailColor: Color {
            switch style {
            case .default: PinzUIAsset.textSecondary.swiftUIColor
            case .destructive: PinzUIAsset.accentRed.swiftUIColor
            }
        }

        public init(
            id: String = UUID().uuidString,
            title: String,
            values: [Value] = [],
            icon: Icon,
            trailIcon: Icon?,
            style: Style = .default,
            action: SettingAction
        ) {
            self.id = id
            self.title = title
            self.values = values
            self.icon = icon
            self.trailIcon = trailIcon
            self.style = style
            self.action = action
        }

        var view: some View {
            HStack(spacing: 8) {
                Group {
                    Image(systemName: icon.rawValue)
                    Text(title)
                }.roundedFount(size: 14, foregroundColor: titleColor)

                Spacer()

                Group {
                    ForEach(values) { value in
                        HStack(spacing: 2) {
                            switch value {
                            case let .text(text):
                                Text(text)
                                    .roundedFount(size: 12, weight: .bold, foregroundColor: trailColor)
                            case let .icon(icon, color):
                                Image(systemName: icon.rawValue)
                                    .foregroundStyle(color)
                                    .roundedFount(size: 14, foregroundColor: trailColor)
                            }
                        }
                    }

                    if let trailIcon {
                        Image(systemName: trailIcon.rawValue)
                            .roundedFount(size: 10, weight: .bold, foregroundColor: trailColor)
                    }
                }
            }
            .frame(height: 48)
        }
    }

    public struct TextFieldSetting {

        public enum Style {
            case `default`
            case multiline
        }

        let id: String
        @Binding
        var text: String
        let placeholder: String
        let style: Style

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
                .frame(maxWidth: .infinity, minHeight: 48)
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
