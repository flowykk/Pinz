public struct SettingsGroup {

    public let title: String?
    public let settings: [Setting]
    public let subtitle: String?
    public let subtitleStyle: SettingSubtitle.Style

    public init(
        title: String? = nil,
        settings: [Setting],
        subtitle: String? = nil,
        subtitleStyle: SettingSubtitle.Style = .default
    ) {
        self.title = title
        self.settings = settings
        self.subtitle = subtitle
        self.subtitleStyle = subtitleStyle
    }
}
