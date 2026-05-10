import Foundation

public struct DeviceTokenRegisterResponseDTO: Codable {
    public let tokenId: String

    enum CodingKeys: String, CodingKey {
        case tokenId = "token_id"
    }

    public init(tokenId: String) {
        self.tokenId = tokenId
    }
}

public struct DeviceTokenUnregisterResponseDTO: Codable {
    public let success: Bool

    public init(success: Bool) {
        self.success = success
    }
}
