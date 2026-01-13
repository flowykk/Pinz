import SwiftUI
import PinzUI

public struct SettingsView: View {

    @State
    public var text: String = ""

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
                    values: [.icon("lock.open", .red), .icon("star.fill", .green)],
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
