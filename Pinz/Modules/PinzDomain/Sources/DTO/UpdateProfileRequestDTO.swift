import Foundation

public struct UpdateProfileRequestDTO: Codable {
    public let username: String

    public init(username: String) {
        self.username = username
    }

    enum CodingKeys: String, CodingKey {
        case username
    }
}
