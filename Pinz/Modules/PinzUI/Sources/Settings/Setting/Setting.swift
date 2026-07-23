import Foundation
import SwiftUI
import PinzDomain

public enum Setting {

    public protocol Icon {
        var rawValue: String { get }
    }

    public enum Action {
        case plain(() -> Void)
        case async(() async throws -> Void)
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

    public struct Title {
        public let title: String
        public let subtitle: String?

        public init(title: String, subtitle: String?) {
            self.title = title
            self.subtitle = subtitle
        }
    }

    public enum Leading {
        case iconTitle(Icon, Title)
        case title(Title)
        case imageTitle(UIImage, Title)

        static public func iconTitle(_ icon: Icon, _ title: String) -> Self {
            return .iconTitle(icon, Title(title: title, subtitle: nil))
        }

        static public func title(_ title: String) -> Self {
            return .title(Title(title: title, subtitle: nil))
        }

        static public func imageTitle(_ image: UIImage, _ title: String) -> Self {
            return .imageTitle(image, Title(title: title, subtitle: nil))
        }
    }

    public enum Trailing {
        case values([Value])
        case icon(Icon, Color? = nil)
        case valuesIcon([Value], Icon)
        case toggle(Binding<Bool>)
    }

    public struct DefaultSetting {

        public enum Style {
            case `default`
            case destructive
        }

        let id: String
        let leading: Leading
        let trailing: Trailing?
        let style: Style
        let action: Action?

        public init(
            id: String = UUID().uuidString,
            leading: Leading,
            trailing: Trailing? = nil,
            style: Style = .default,
            action: Action? = nil
        ) {
            self.id = id
            self.leading = leading
            self.trailing = trailing
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

    public struct PickerSetting {

        let id: String
        let leading: Leading
        let value: Value?
        var isPickerPresented: Binding<Bool>

        public init(
            id: String,
            leading: Leading,
            value: Value? = nil,
            isPickerPresented: Binding<Bool>
        ) {
            self.id = id
            self.leading = leading
            self.value = value
            self.isPickerPresented = isPickerPresented
        }
    }

    enum PickerIcon: String, Icon {
        case chevrons = "chevron.up.chevron.down"
    }

    public struct ToggleSetting {

        let id: String
        let leading: Leading
        var value: Binding<Bool>

        public init(
            id: String,
            leading: Leading,
            value: Binding<Bool>,
        ) {
            self.id = id
            self.leading = leading
            self.value = value
        }
    }

    case `default`(DefaultSetting)
    case textField(TextFieldSetting)

    public static func picker(_ setting: PickerSetting) -> Self {
        .default(DefaultSetting(
            id: setting.id,
            leading: setting.leading,
            trailing: .valuesIcon(setting.value.flatMap { [$0] } ?? [], PickerIcon.chevrons),
            action: .plain { setting.isPickerPresented.wrappedValue = true }
        ))
    }

    public static func toggle(_ setting: ToggleSetting) -> Self {
        .default(DefaultSetting(
            id: setting.id,
            leading: setting.leading,
            trailing: .toggle(setting.value),
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
