import SwiftUI

public struct PinzButton: View {

    public enum ButtonType {
        case icon(IconType)
        case text(String)
    }

    public enum IconType: String {
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
            Group {
                switch type {
                case let .icon(icon):
                    Image(systemName: icon.rawValue)
                        .roundedFount(size: 20, weight: .bold)
                        .frame(width: 40, height: 40)
                        .tint(tint)
                case let .text(text):
                    Text(text)
                        .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                }
            }
            .tint(tint)
        }
    }
}
