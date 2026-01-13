import SwiftUI
import PinzUI

public enum PrivacyStatus: String, CaseIterable, SegmentedItem {
    case `private`
    case `public`

    public var id: String { rawValue }
    public var title: String {
        switch self {
        case .private: "lock.open.fill"
        case .public: "lock.fill"
        }
    }
}

public struct SettingsView: View {

    @State
    public var text: String = ""

    @State
    public var privacyStatus: PrivacyStatus = .private

    public init() {}

    public var body: some View {
        SettingsGroup(
            title: "Общая информация",
            settings: [
                .default(.init(
                    title: "Сезон",
                    values: [.text("Лето")],
                    icon: "sun.max.fill",
                    style: .default,
                    action: .plain { print("smth") }
                )),
                .default(.init(
                    title: "Даты",
                    values: [.text("22.08.2025 - 31.08.2025")],
                    icon: "calendar",
                    style: .default,
                    action: .plain { print("smth") }
                )),
                .default(.init(
                    title: "Тестирую",
                    values: [.icon(privacyStatus.title, .red), .icon("star.fill", .green)],
                    icon: "square.fill",
                    style: .default,
                    action: .plain { print("smth") }
                )),
                .textField(.init(
                    id: "textFieldInput",
                    text: $text,
                    placeholder: "123",
                    style: .multiline
                ))
            ],
            subtitle: "Путешествие не будет отображаться в общей ленте"
        )
        .padding(.horizontal, 12)
    }
}
