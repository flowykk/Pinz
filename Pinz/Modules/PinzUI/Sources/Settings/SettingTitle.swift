import SwiftUI

public struct SettingTitle: View {

    private let text: String

    public init(_ text: String) {
        self.text = text
    }

    public var body: some View {
        Text(text)
            .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
    }
}
