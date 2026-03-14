import Foundation

public struct SubmitEmailResponse: Codable {
    public let isRegistered: Bool
    public let registrationId: String?

    enum CodingKeys: String, CodingKey {
        case isRegistered = "is_registered"
        case registrationId = "registration_id"
    }
}
