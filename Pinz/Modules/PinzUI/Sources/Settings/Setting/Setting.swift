import Foundation
import SwiftUI

public enum Setting {

    public protocol Icon {
        var rawValue: String { get }
    }

    public enum Action {
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
        let icon: Icon?
        let trailIcon: Icon?
        let style: Style
        let action: Action

        public init(
            id: String = UUID().uuidString,
            title: String,
            values: [Value] = [],
            icon: Icon? = nil,
            trailIcon: Icon? = nil,
            style: Style = .default,
            action: Action
        ) {
            self.id = id
            self.title = title
            self.values = values
            self.icon = icon
            self.trailIcon = trailIcon
            self.style = style
            self.action = action
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
            id: String,
            text: Binding<String>,
            placeholder: String,
            style: Style = .default
        ) {
            self.id = id
            self._text = text
            self.placeholder = placeholder
            self.style = style
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
