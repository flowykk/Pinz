import SwiftUI
import PinzDomain

struct AddTagView: View {

    @State private var tag: String = ""
    @FocusState private var focused: Bool
    let onTagAdd: ((MediaTag) -> Void)?

    @Environment(\.dismiss) var dismiss

    var body: some View {
        VStack(spacing: 8) {
            Spacer()

            SettingsGroup(
                settings: [
                    .textField(Setting.TextFieldSetting(
                        id: "newTagTextField",
                        text: $tag,
                        placeholder: "Тег",
                        focused: $focused,
                    )),
                ]
            )

            PinzButton(
                type: .slot(style: .primary, title: "Готово"),
                action: .plain {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        onTagAdd?(MediaTag(tag: tag))
                    }
                    dismiss()
                }
            )

            Spacer(minLength: 8)
        }
        .onAppear { focused = true }
        .padding(.horizontal, 12)
    }
}
