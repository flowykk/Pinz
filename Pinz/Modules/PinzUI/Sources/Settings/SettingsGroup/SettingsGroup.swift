public struct SettingsGroup {

    public let title: String?
    public let settings: [Setting]
    public let subtitle: String?

    public init(
        title: String? = nil,
        settings: [Setting],
        subtitle: String? = nil
    ) {
        self.title = title
        self.settings = settings
        self.subtitle = subtitle
    }
}
