import Foundation
import SwiftUI
import PinzDomain
import PinzDomain

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
        let icon: Icon?
        let values: [Value]
        let trailIcon: Icon?
        let style: Style
        let action: Action?

        public init(
            id: String = UUID().uuidString,
            title: String,
            icon: Icon? = nil,
            values: [Value] = [],
            trailIcon: Icon? = nil,
            style: Style = .default,
            action: Action? = nil
        ) {
            self.id = id
            self.title = title
            self.icon = icon
            self.values = values
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
        @Binding var text: String
        let placeholder: String
        var focused: FocusState<Bool>.Binding?
        let style: Style

        public init(
            id: String,
            text: Binding<String>,
            placeholder: String,
            focused: FocusState<Bool>.Binding? = nil,
            style: Style = .default
        ) {
            self.id = id
            self._text = text
            self.placeholder = placeholder
            self.focused = focused
            self.style = style
        }
    }

    public struct PickerSetting<Item: PickerItem> {

        let id: String
        let items: [Item]
        let title: String
        let value: Binding<Item>
        let icon: Icon?
        let action: Action?

        public init(
            id: String,
            items: [Item],
            title: String,
            value: Binding<Item>,
            icon: Icon? = nil,
            action: Action? = nil
        ) {
            self.id = id
            self.items = items
            self.title = title
            self.value = value
            self.icon = icon
            self.action = action
        }
    }

    enum PickerIcon: String, Icon {
        case chevrons = "chevron.up.chevron.down"
    }

    case `default`(DefaultSetting)
    case textField(TextFieldSetting)

    public static func picker<Item: PickerItem>(_ setting: PickerSetting<Item>) -> Self {
        .default(DefaultSetting(
            id: setting.id,
            title: setting.title,
            icon: setting.icon,
            values: [],
            trailIcon: PickerIcon.chevrons,
            action: setting.action
        ))
    }
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
