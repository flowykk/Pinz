import SwiftUI

public struct LoadingView: View {

    private let status: String?

    public init(
        status: String? = nil
    ) {
        self.status = status
    }

    public var body: some View {
        VStack(spacing: 12) {
            ProgressView()
            if let status {
                Text(status)
                    .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    .id(status)
                    .transition(.opacity)
            }
        }
        .animation(.easeInOut(duration: 0.3), value: status)
    }
}
