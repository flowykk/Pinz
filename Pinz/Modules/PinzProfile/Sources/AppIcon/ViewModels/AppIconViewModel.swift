import Foundation
import SwiftUI

@MainActor @Observable
public final class AppIconViewModel: Identifiable {

    enum Intent {
        case change
    }

    let name: String
    let task: String?
    var selected: Bool

    public static let appIconKey = "pinzAppIcon"
    public static let defaultAppIcon = "PinzLightP"

    private let userDefaults = UserDefaults.standard

    public init(appIcon: String, task: String? = nil) {
        self.name = appIcon
        self.task = task
        if let savedIcon = userDefaults.string(forKey: Self.appIconKey) {
            selected = appIcon == savedIcon ? true : false
        } else {
            selected = appIcon == Self.defaultAppIcon ? true : false
        }
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .change:
            withAnimation(.easeInOut(duration: 0.3)) {
                selected = name == userDefaults.string(forKey: Self.appIconKey)
            }
        }
    }
}
