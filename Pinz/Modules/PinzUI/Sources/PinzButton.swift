import SwiftUI

public struct PinzButton: View {
    public enum ButtonType: String {
        case chevronLeft = "chevron.left"

        case personAdd = "person.fill.badge.plus"
        case pencil = "pencil"
    }

    private let type: ButtonType
    private let tint: Color
    private let action: () -> Void

    public init(
        type: ButtonType,
        tint: Color = .black,
        action: @escaping () -> Void = {}
    ) {
        self.type = type
        self.tint = tint
        self.action = action
    }

    public var body: some View {
        Button(action: action) {
            Image(systemName: type.rawValue)
                .font(.system(size: 20, weight: .bold))
                .frame(width: 40, height: 40)
                .tint(tint)
        }
    }
}
