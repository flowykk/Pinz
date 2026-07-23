public struct AddMediaConfirmDTO: Codable {
    public let status: String
    public let alreadyConfirmed: Bool

    public init(status: String, alreadyConfirmed: Bool) {
        self.status = status
        self.alreadyConfirmed = alreadyConfirmed
    }

    enum CodingKeys: String, CodingKey {
        case status
        case alreadyConfirmed = "already_confirmed"
    }
}
