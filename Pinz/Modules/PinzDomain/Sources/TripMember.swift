import Foundation

public struct TripMember: Identifiable {
    public let id: UUID = UUID()
    public let isPrivate: Bool
    public let username: String

    public init(isPrivate: Bool, username: String) {
        self.isPrivate = isPrivate
        self.username = username
    }
}
