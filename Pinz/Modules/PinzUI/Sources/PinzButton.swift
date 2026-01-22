import SwiftUI

public struct PinzButton: View {

    public enum SlotStyle: Equatable {
        case primary
        case secondary(needBorder: Bool = false)

        var backgroundColor: Color {
            switch self {
            case .primary:
                return PinzUIAsset.textPrimary.swiftUIColor
            case .secondary:
                return PinzUIAsset.backgroundSecondary.swiftUIColor
            }
        }

        var textColor: Color {
            switch self {
            case .primary:
                return PinzUIAsset.backgroundSecondary.swiftUIColor
            case .secondary:
                return PinzUIAsset.textPrimary.swiftUIColor
            }
        }
    }

    public enum ButtonType {
        case icon(IconType)
        case text(String)
        case slot(style: SlotStyle, title: String)
    }

    public enum IconType: String {
        case chevronLeft = "chevron.left"
        case xmark = "xmark"

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
                        .padding(.horizontal, 6)
                case let .slot(style, title):
                    HStack {
                        Spacer()
                        Text(title)
                            .roundedFount(size: 16, foregroundColor: style.textColor)
                            .frame(height: 52)
                        Spacer()
                    }
                    .background(style.backgroundColor)
                    .cornerRadius(26)
                    .if(style == .secondary(needBorder: true)) { view in
                        return view.overlay(
                            RoundedRectangle(cornerRadius: 26)
                                .stroke(PinzUIAsset.textTertiary.swiftUIColor.opacity(0.6), lineWidth: 3)
                        )
                    }
                }
            }
        }
    }
}
