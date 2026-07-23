public struct DeletePinResponseDTO: Codable {
    public let deletionMode: String

    public init(deletionMode: String) {
        self.deletionMode = deletionMode
    }

    enum CodingKeys: String, CodingKey {
        case deletionMode = "deletion_mode"
    }
}
