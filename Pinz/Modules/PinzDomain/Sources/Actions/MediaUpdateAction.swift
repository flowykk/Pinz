import Foundation

public struct MediaUpdateAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (MediaItem) -> Void

    public init(action: @escaping (MediaItem) -> Void) {
        self.action = action
    }

    public static func == (lhs: MediaUpdateAction, rhs: MediaUpdateAction) -> Bool {
        lhs.id == rhs.id
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
