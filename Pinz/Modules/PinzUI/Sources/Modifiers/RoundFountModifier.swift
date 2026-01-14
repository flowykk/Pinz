import SwiftUI

public struct RoundFontModifier: ViewModifier {
    private let size: CGFloat
    private let weight: Font.Weight
    private let foregroundColor: Color

    public init(
        size: CGFloat,
        weight: Font.Weight = .medium,
        foregroundColor: Color = PinzUIAsset.textPrimary.swiftUIColor
    ) {
        self.size = size
        self.weight = weight
        self.foregroundColor = foregroundColor
    }

    public func body(content: Content) -> some View {
        content
            .font(.system(size: size, weight: weight, design: .rounded))
            .foregroundStyle(foregroundColor)
    }
}

extension View {
    public func roundedFount(
        size: CGFloat,
        weight: Font.Weight = .medium,
        foregroundColor: Color = PinzUIAsset.textPrimary.swiftUIColor
    ) -> some View {
        self.modifier(RoundFontModifier(size: size, weight: weight, foregroundColor: foregroundColor))
    }
}
