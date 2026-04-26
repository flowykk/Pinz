public struct TripSettingsDTO: Codable {
    public let notificationsEnabled: Bool?
    public let privacyLevel: String?

    enum CodingKeys: String, CodingKey {
        case notificationsEnabled = "notifications_enabled"
        case privacyLevel = "privacy_level"
    }
}
