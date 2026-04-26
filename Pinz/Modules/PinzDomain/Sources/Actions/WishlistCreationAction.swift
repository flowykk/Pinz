import Foundation

public struct WishlistCreationAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (DesiredPlace) -> Void

    public init(action: @escaping (DesiredPlace) -> Void) {
        self.action = action
    }

    public static func == (lhs: WishlistCreationAction, rhs: WishlistCreationAction) -> Bool {
        lhs.id == rhs.id
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
