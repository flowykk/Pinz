import SwiftUI

public struct SettingSubtitle: View {

    private let text: String

    public init(_ text: String) {
        self.text = text
    }

    public var body: some View {
        Text(text)
            .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
    }
}
