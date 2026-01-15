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
            AppIconViewModel(appIcon: "RewindPink"),
            AppIconViewModel(appIcon: "RewindGradient"),
            AppIconViewModel(appIcon: "RewindForest"),
            AppIconViewModel(appIcon: "RewindLazer"),
            AppIconViewModel(appIcon: "RewindSakura"),
            AppIconViewModel(appIcon: "RewindSea"),
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
