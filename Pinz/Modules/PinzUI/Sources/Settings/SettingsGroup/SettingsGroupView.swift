import SwiftUI

extension SettingsGroup: View {
    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let title {
                Text(title)
                    .font(.system(size: 16, weight: .medium, design: .rounded))
                    .padding(.bottom, 6)
                    .padding(.leading, 12)
            }

            VStack(spacing: 0) {
                ForEach(settings) { setting in
                    setting.view
                }
                .padding(.horizontal, 16)
            }
            .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
            .cornerRadius(26)

            if let subtitle {
                Text(subtitle)
                    .font(.system(size: 12, weight: .medium, design: .rounded))
                    .foregroundStyle(PinzUIAsset.textSecondary.swiftUIColor)
                    .padding(.top, 4)
                    .padding(.leading, 12)
            }
        }
    }
}
