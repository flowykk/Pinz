import SwiftUI

public struct PinzButton: View {

    public enum ButtonType {
        case icon(IconType)
        case text(String)
        case slot(String, Color)
    }

    public enum IconType: String {
        case chevronLeft = "chevron.left"
        case xmark = "xmark"

        case personAdd = "person.fill.badge.plus"
        case pencil = "pencil"
    }

    public struct Slot {
        let title: String
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
                case let .slot(title, color):
                    HStack {
                        Spacer()
                        Text(title)
                            .roundedFount(size: 16, foregroundColor: tint)
                            .frame(height: 52)
                        Spacer()
                    }
                    .background(color)
                    .cornerRadius(26)
                }
            }
            .tint(tint)
        }
    }
}
