public struct SuccessResponse: Codable {
    public let success: Bool

    public init(success: Bool) {
        self.success = success
    }
}

