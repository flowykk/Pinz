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
        case ellipsis = "ellipsis"
        case plus = "plus"

        case personAdd = "person.fill.badge.plus"
        case pencil = "pencil"

        case stories = "rectangle.portrait.on.rectangle.portrait.angled"
        case warning = "exclamationmark.triangle.fill"
        case checkmark = "checkmark.circle.fill"

        case download = "square.and.arrow.down.fill"
        case crop = "crop"
    }

    public enum Action {
        case plain(() -> Void)
        case async(() async throws -> Void)
    }

    private let type: ButtonType
    private let tint: Color
    private let action: Action
    private let disabled: Bool?

    @State private var isLoading: Bool = false

    public init(
        type: ButtonType,
        tint: Color = .black,
        disabled: Bool? = false,
        action: Action
    ) {
        self.type = type
        self.tint = tint
        self.disabled = disabled
        self.action = action
    }

    public var body: some View {
        Button {
            switch action {
            case let .plain(action):
                action()
            case let .async(action):
                isLoading = true
                Task {
                    try await action()
                    isLoading = false
                }
            }
        } label: {
            Group {
                switch type {
                case let .icon(icon):
                    Image(systemName: icon.rawValue)
                        .roundedFont(size: 20, weight: .semibold, foregroundColor: tint)
                        .frame(width: 40, height: 40)
                        .contentShape(Rectangle())
                case let .text(text):
                    Text(text)
                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .padding(.horizontal, 6)
                case let .slot(style, title):
                    HStack {
                        Spacer()
                        if isLoading {
                            ProgressView()
                                .tint(style.textColor)
                        } else {
                            Text(title)
                                .roundedFont(size: 16, foregroundColor: style.textColor)
                        }
                        Spacer()
                    }
                    .frame(height: 52)
                    .background(style.backgroundColor)
                    .background(.ultraThinMaterial)
                    .cornerRadius(26)
                    .if(style == .secondary(needBorder: true)) { view in
                        return view.overlay(
                            RoundedRectangle(cornerRadius: 26)
                                .stroke(PinzUIAsset.textPrimary.swiftUIColor, lineWidth: 1)
                        )
                    }
                }
            }
        }
        .buttonStyle(.plain)
        .ifLet(disabled, apply: { view, disabled in
            view
                .disabledWithOpacity(disabled)
                .animation(.easeInOut(duration: 0.2), value: disabled)
        })
    }
}
