import Foundation

public struct PinDeleteAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (Pin) -> Void

    public init(action: @escaping (Pin) -> Void) {
        self.action = action
    }

    public static func == (lhs: PinDeleteAction, rhs: PinDeleteAction) -> Bool {
        lhs.id == rhs.id
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
