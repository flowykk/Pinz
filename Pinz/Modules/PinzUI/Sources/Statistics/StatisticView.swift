import SwiftUI

public struct StatisticView: View {
    private let icon: String
    private let text: String?
    private let iconColor: PinzUIColors

    public init (
        icon: String,
        text: String? = nil,
        iconColor: PinzUIColors = PinzUIAsset.textSecondary
    ) {
        self.icon = icon
        self.text = text
        self.iconColor = iconColor
    }

    public var body: some View {
        HStack(spacing: 2) {
            Group {
                Image(systemName: icon)
                if let text {
                    Text(text)
                }
            }.roundedFount(size: 14, foregroundColor: iconColor.swiftUIColor)
        }
    }
}
