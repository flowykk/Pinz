import Foundation
import SwiftUI

extension PinzUIAsset {
    public static var mapOverlayTextColor: Color {
        let raw = UserDefaults.standard.string(forKey: PinzMapStyle.mapStyleKey)
        let style = raw.flatMap(PinzMapStyle.init(rawValue:)) ?? .satelight
        switch style {
        case .scheme:
            return textPrimary.swiftUIColor
        case .satelight, .hybrid:
            return .white
        }
    }
}
