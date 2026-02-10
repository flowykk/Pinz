import SwiftUI

public struct DescriptionEditingView: View {
    
    private let title: String
    private let placeholder: String
    @Binding private var text: String
    
    public init(
        title: String = "Описание",
        text: Binding<String>,
        placeholder: String = "Описание путешествия"
    ) {
        self.title = title
        self._text = text
        self.placeholder = placeholder
    }
    
    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle(title)
                .padding(.bottom, 6)
                .padding(.leading, 12)
            
            SettingsGroup(settings: [
                .textField(Setting.TextFieldSetting(
                    id: "descriptionEditingTextField",
                    text: $text,
                    placeholder: placeholder,
                    style: .multiline
                ))
            ])
        }
    }
}
