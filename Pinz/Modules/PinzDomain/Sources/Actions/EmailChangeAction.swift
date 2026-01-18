import Foundation

public struct EmailChangeAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (String) -> Void

    public init(action: @escaping (String) -> Void) {
        self.action = action
    }

    public static func == (lhs: EmailChangeAction, rhs: EmailChangeAction) -> Bool {
        lhs.id == rhs.id
    }
    
    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
