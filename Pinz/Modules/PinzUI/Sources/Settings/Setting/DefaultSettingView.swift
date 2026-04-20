import SwiftUI

extension Setting.DefaultSetting {
    var titleColor: Color {
        switch style {
        case .default: PinzUIAsset.textPrimary.swiftUIColor
        case .destructive: PinzUIAsset.accentRed.swiftUIColor
        }
    }

    var subtitleColor: Color {
        switch style {
        case .default: PinzUIAsset.textSecondary.swiftUIColor
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

            leadingView

            Spacer()

            trailingView
        }
        .frame(height: 52)
        .contentShape(Rectangle())
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

// MARK: SettingTrailingView

extension Setting.DefaultSetting {
    @ViewBuilder
    private var trailingView: some View {
        if let trailing {
            switch trailing {
            case let .icon(icon, color):
                trailingIconView(for: icon, with: color)

            case let .values(values):
                valueView(for: values)

            case let .valuesIcon(values, icon):
                valueView(for: values)
                    .padding(.trailing, 8)
                trailingIconView(for: icon)
            case let .toggle(value):
                toggleView(with: value)
            }
        } else {
            EmptyView()
        }
    }

    private func valueView(for values: [Setting.Value]) -> some View {
        ForEach(values) { value in
            HStack(spacing: 2) {
                switch value {
                case let .text(text):
                    Text(text)
                        .roundedFont(size: 12, foregroundColor: trailColor)
                case let .icon(icon, color):
                    trailingIconView(for: icon, with: color)
                }
            }
        }
    }

    private func trailingIconView(
        for icon: Setting.Icon,
        with color: Color? = nil
    ) -> some View {
        Image(systemName: icon.rawValue)
            .roundedFont(size: 12, foregroundColor: color ?? trailColor)
    }

    private func toggleView(with value: Binding<Bool>) -> some View {
        Toggle(isOn: value) {}
    }
}

// MARK: SettingLeadingView

extension Setting.DefaultSetting {
    @ViewBuilder
    private var leadingView: some View {
        switch leading {
        case let .iconTitle(icon, title):
            HStack(spacing: 0) {
                iconView(for: icon)
                    .padding(.trailing, 12)
                titleView(for: title)
            }

        case let .title(title):
            titleView(for: title)

        case let .imageTitle(image, title):
            HStack(spacing: 0) {
                imageView(for: image)
                    .padding(.leading, -6)
                    .padding(.trailing, 12)
                titleView(for: title)
            }
        }
    }

    private func titleView(for title: Setting.Title) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(title.title)
                .roundedFont(size: 16, foregroundColor: titleColor)
            if let subtitle = title.subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .roundedFont(size: 12, foregroundColor: subtitleColor)
            }
        }
    }

    private func imageView(for image: UIImage) -> some View {
        Image(uiImage: image)
            .resizable()
            .scaledToFill()
            .frame(36)
            .cornerRadius(18)
    }

    private func iconView(
        for icon: Setting.Icon,
        with color: Color? = nil
    ) -> some View {
        Image(systemName: icon.rawValue)
            .roundedFont(size: 18, foregroundColor: color ?? titleColor)
            .frame(16)
    }
}
