import Foundation

public struct PinResponseAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (PinResponseDTO) -> Void

    public init(action: @escaping (PinResponseDTO) -> Void) {
        self.action = action
    }

    public static func == (lhs: PinResponseAction, rhs: PinResponseAction) -> Bool {
        lhs.id == rhs.id
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
