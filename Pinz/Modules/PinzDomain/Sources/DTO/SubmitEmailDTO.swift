import Foundation

public struct SubmitEmailDTO: Codable {
    public let isRegistered: Bool
    public let registrationId: String?

    public init(isRegistered: Bool, registrationId: String?) {
        self.isRegistered = isRegistered
        self.registrationId = registrationId
    }

    enum CodingKeys: String, CodingKey {
        case isRegistered = "is_registered"
        case registrationId = "registration_id"
    }
}
