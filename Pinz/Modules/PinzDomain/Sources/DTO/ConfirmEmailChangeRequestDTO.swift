import Foundation

public struct ConfirmEmailChangeRequestDTO: Codable {
    public let verificationCode: String

    public init(verificationCode: String) {
        self.verificationCode = verificationCode
    }

    enum CodingKeys: String, CodingKey {
        case verificationCode = "verification_code"
    }
}
