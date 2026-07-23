public struct AddMediaTakeoverDTO: Codable {
    public let isInitiator: Bool
    public let currentInitiator: PublicUserProfileDTO?
    public let takeoverAvailableAt: String?

    public init(
        isInitiator: Bool,
        currentInitiator: PublicUserProfileDTO? = nil,
        takeoverAvailableAt: String? = nil
    ) {
        self.isInitiator = isInitiator
        self.currentInitiator = currentInitiator
        self.takeoverAvailableAt = takeoverAvailableAt
    }

    enum CodingKeys: String, CodingKey {
        case isInitiator = "is_initiator"
        case currentInitiator = "current_initiator"
        case takeoverAvailableAt = "takeover_available_at"
    }
}
