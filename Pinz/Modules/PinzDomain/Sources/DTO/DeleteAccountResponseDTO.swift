import Foundation

public struct DeleteAccountResponseDTO: Codable {
    public let success: Bool

    public init(success: Bool) {
        self.success = success
    }

    enum CodingKeys: String, CodingKey {
        case success
    }
}
