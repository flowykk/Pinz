import SwiftUI

public struct StatisticView: View {
    let text: String
    let icon: String

    public init (text: String, icon: String) {
        self.text = text
        self.icon = icon
    }

    public var body: some View {
        HStack(spacing: 6) {
            Group {
                Image(systemName: icon)
                Text(text)
            }.roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
        }
    }
}
