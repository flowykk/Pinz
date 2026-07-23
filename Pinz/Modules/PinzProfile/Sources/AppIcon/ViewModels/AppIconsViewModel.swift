import SwiftUI

@MainActor @Observable
public final class AppIconsViewModel: Identifiable {
    
    enum Intent {
        case change(icon: AppIconViewModel)
    }

    public var appIcons: [AppIconViewModel]

    public init() {
        appIcons = [
            AppIconViewModel(appIcon: AppIconViewModel.defaultAppIcon),
            AppIconViewModel(appIcon: "PinzLight"),
            AppIconViewModel(appIcon: "PinzMountainsWhite"),
            AppIconViewModel(appIcon: "PinzMountainsBlue"),
            AppIconViewModel(appIcon: "PinzPin"),
            AppIconViewModel(appIcon: "PinzTransport"),
        ]
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .change(let icon):
            changeAppIcon(icon: icon)
        }
    }

    private func changeAppIcon(icon: AppIconViewModel) {
        guard UIApplication.shared.supportsAlternateIcons else { return }

        if icon.name == AppIconViewModel.defaultAppIcon {
            UIApplication.shared.setAlternateIconName(nil)
        }
        UIApplication.shared.setAlternateIconName(icon.name)

        UserDefaults.standard.set(icon.name, forKey: AppIconViewModel.appIconKey)
        for icon in self.appIcons {
            icon.dispatch(.change)
        }
    }
}
