import SwiftUI

public struct SettingsGroup: View {

    public let title: String?
    public let settings: [Setting]
    public let subtitle: String?

    public init(
        title: String?,
        settings: [Setting],
        subtitle: String?
    ) {
        self.title = title
        self.settings = settings
        self.subtitle = subtitle
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let title {
                Text(title)
                    .font(.system(size: 14, weight: .medium, design: .rounded))
                    .padding(.bottom, 6)
                    .padding(.leading, 4)
            }

            VStack(spacing: 0) {
                ForEach(settings) { setting in
                    setting.view
                }
                .padding(.horizontal, 10)
            }
            .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
            .cornerRadius(22)

            if let subtitle {
                Text(subtitle)
                    .font(.system(size: 12, weight: .medium, design: .rounded))
                    .foregroundStyle(PinzUIAsset.textSecondary.swiftUIColor)
                    .padding(.top, 2)
                    .padding(.leading, 4)
            }
        }
    }
}
