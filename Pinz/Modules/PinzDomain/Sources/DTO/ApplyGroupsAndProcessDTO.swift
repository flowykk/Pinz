public struct ApplyGroupsAndProcessDTO: Codable {
    public let message: String
    public let status: String

    public init(message: String, status: String) {
        self.message = message
        self.status = status
    }
}
