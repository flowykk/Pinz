public struct PrivacyResponseDTO: Codable {
    public let privacyLevel: String

    public init(privacyLevel: String) {
        self.privacyLevel = privacyLevel
    }

    enum CodingKeys: String, CodingKey {
        case privacyLevel = "privacy_level"
    }
}
