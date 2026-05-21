import SwiftUI

public struct DescriptionEditingView: View {

    private let title: String?
    private let subtitle: String?
    private let subtitleStyle: SettingSubtitle.Style
    private let placeholder: String
    private let textFieldId: String
    @Binding private var text: String

    public init(
        title: String? = nil,
        subtitle: String? = nil,
        subtitleStyle: SettingSubtitle.Style = .default,
        text: Binding<String>,
        placeholder: String,
        textFieldId: String = "descriptionEditingTextField"
    ) {
        self.title = title
        self.subtitle = subtitle
        self.subtitleStyle = subtitleStyle
        self._text = text
        self.placeholder = placeholder
        self.textFieldId = textFieldId
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let title {
                SettingTitle(title)
                    .padding(.bottom, 6)
                    .padding(.leading, 12)
            }

            SettingsGroup(
                settings: [
                    .textField(Setting.TextFieldSetting(
                        id: textFieldId,
                        text: $text,
                        placeholder: placeholder,
                        style: .multiline
                    ))
                ],
                subtitle: subtitle,
                subtitleStyle: subtitleStyle
            )
        }
    }
}
