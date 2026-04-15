import Foundation

public struct ChangeEmailRequestDTO: Codable {
    public let email: String

    public init(email: String) {
        self.email = email
    }

    enum CodingKeys: String, CodingKey {
        case email
    }
}
