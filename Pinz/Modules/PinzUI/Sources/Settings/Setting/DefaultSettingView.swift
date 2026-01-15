import SwiftUI

extension Setting.DefaultSetting {
    var titleColor: Color {
        switch style {
        case .default: PinzUIAsset.textPrimary.swiftUIColor
        case .destructive: PinzUIAsset.accentRed.swiftUIColor
        }
    }

    var trailColor: Color {
        switch style {
        case .default: PinzUIAsset.textSecondary.swiftUIColor
        case .destructive: PinzUIAsset.accentRed.swiftUIColor
        }
    }

    var view: some View {
        Button {
            switch action {
            case let .async(action):
                Task {
                    try await action()
                }
            case let .plain(action):
                action()
            }
        } label: {
            settingView
                .frame(height: 52)
        }
    }

    private var settingView: some View {
        HStack(spacing: 0) {
            if let icon {
                Image(systemName: icon.rawValue)
                    .roundedFount(size: 18, foregroundColor: titleColor)
                    .frame(width: 16, height: 16)
                    .padding(.trailing, 12)
            }
            Text(title)
                .roundedFount(size: 16, foregroundColor: titleColor)

            Spacer()

            ForEach(values) { value in
                HStack(spacing: 2) {
                    switch value {
                    case let .text(text):
                        Text(text)
                            .roundedFount(size: 12, foregroundColor: trailColor)
                    case let .icon(icon, color):
                        Image(systemName: icon.rawValue)
                            .foregroundStyle(color)
                            .roundedFount(size: 12, foregroundColor: trailColor)
                    }
                }
            }.padding(.trailing, 6)

            if let trailIcon {
                Image(systemName: trailIcon.rawValue)
                    .roundedFount(size: 12, foregroundColor: trailColor)
            }
        }
    }
}
