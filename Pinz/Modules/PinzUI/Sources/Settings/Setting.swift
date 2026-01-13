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

        public enum Value: Identifiable {
            case text(String)
            case icon(String, Color)

            public var id: String {
                switch self {
                case .text(let str): return "string_\(str)"
                case .icon(let name): return "icon_\(name)"
                }
            }
        }

        public let id: String
        public let title: String
        public let values: [Value]
        public let icon: String
        public let trailIcon: String?
        public let style: Style
        public let action: SettingAction

        public init(
            id: String = UUID().uuidString,
            title: String,
            values: [Value] = [],
            icon: String,
            trailIcon: String? = nil,
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
                    Image(systemName: icon)
                        .modifier(RoundFontModifier(size: 18, weight: .medium))
                    Text(title)
                        .modifier(RoundFontModifier(size: 14, weight: .medium))
                }
                .foregroundStyle(PinzUIAsset.textPrimary.swiftUIColor)

                Spacer()

                Group {
                    ForEach(values) { value in
                        HStack(spacing: 2) {
                            switch value {
                            case let .text(text):
                                Text(text)
                                    .modifier(RoundFontModifier(size: 12, weight: .medium))
                            case let .icon(systemName, color):
                                Image(systemName: systemName)
                                    .foregroundStyle(color)
                                    .modifier(RoundFontModifier(size: 14, weight: .medium))
                            }
                        }
                    }

                    if let trailIcon {
                        Image(systemName: trailIcon)
                            .modifier(RoundFontModifier(size: 12, weight: .medium))
                    }
                }
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
