import SwiftUI

extension SettingsGroup: View {
    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let title {
                SettingTitle(title)
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
                SettingSubtitle(subtitle)
                    .padding(.top, 4)
                    .padding(.leading, 12)
            }
        }
    }
}
