import SwiftUI

public struct HeaderTitle: View {

    private let title: String
    private let subtitle: String?

    public init(_ title: String, subtitle: String? = nil) {
        self.title = title
        self.subtitle = subtitle
    }

    public var body: some View {
        VStack(spacing: 0) {
            Text(title)
                .roundedFont(size: 16, weight: .semibold, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
            if let subtitle {
                Text(subtitle)
                    .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
        }
    }
}
