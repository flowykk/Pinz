import SwiftUI

public struct StatisticView: View {
    private let icon: String
    private let text: String?
    private let iconSize: CGFloat
    private let textSize: CGFloat
    private let iconColor: PinzUIColors

public init (
        icon: String,
        text: String? = nil,
        iconSize: CGFloat = 14,
        textSize: CGFloat = 14,
        iconColor: PinzUIColors = PinzUIAsset.textSecondary
    ) {
        self.icon = icon
        self.text = text
        self.iconSize = iconSize
        self.textSize = textSize
        self.iconColor = iconColor
    }

    public var body: some View {
        HStack(spacing: 2) {
            Group {
                Image(systemName: icon)
                    .roundedFont(size: iconSize, foregroundColor: iconColor.swiftUIColor)
                if let text {
                    Text(text)
                        .roundedFont(size: textSize, foregroundColor: iconColor.swiftUIColor)
                }
            }
        }
    }
}
