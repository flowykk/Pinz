import Foundation

public struct RefreshTokenResponse: Codable {
    public let accessToken: String

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
    }
}
