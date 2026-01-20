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

    @ViewBuilder
    var view: some View {
        if let action {
            Button {
                handle(action)
            } label: {
                settingView
            }
        } else {
            settingView
        }
    }

    private var settingView: some View {
        HStack(spacing: 0) {
            if let icon {
                Image(systemName: icon.rawValue)
                    .roundedFount(size: 18, foregroundColor: titleColor)
                    .frame(16)
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
                            .roundedFount(size: 16, foregroundColor: color)
                    }
                }
            }.padding(.trailing, trailIcon != nil ? 6 : 0)

            if let trailIcon {
                Image(systemName: trailIcon.rawValue)
                    .roundedFount(size: 12, foregroundColor: trailColor)
            }
        }.frame(height: 52)
    }

    private func handle(_ action: Setting.Action) {
        switch action {
        case let .async(action):
            Task {
                try await action()
            }
        case let .plain(action):
            action()
        }
    }
}
