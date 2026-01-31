import Foundation
import SwiftUI

public struct TripMember: Hashable, Identifiable {
    public let id: UUID = UUID()
    public let isPrivate: Bool
    public let username: String
    public let avatar: UIImage

    public init(isPrivate: Bool, username: String, avatar: UIImage) {
        self.isPrivate = isPrivate
        self.username = username
        self.avatar = avatar
    }
}
