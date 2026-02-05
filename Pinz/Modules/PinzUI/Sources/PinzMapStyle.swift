import SwiftUI
import MapKit
import Foundation

public enum PinzMapStyle: String, SegmentedItem {
    case scheme
    case hybrid
    case satelight

    public var id: Self { self }
    public static let mapStyleKey = "pinzMapStyle"

    public var content: SegmentedItemContent {
        switch self {
        case .scheme: .text("Схема")
        case .satelight: .text("Спутник")
        case .hybrid: .text("Гибрид")
        }
    }

    public func toMapKitMapStyle() -> MapStyle {
        switch self {
        case .scheme: .standard(elevation: .realistic)
        case .satelight: .imagery(elevation: .realistic)
        case .hybrid: .hybrid(elevation: .realistic)
        }
    }
    
    public static var saved: PinzMapStyle {
        let userDefaults = UserDefaults.standard
        
        if let savedStyle = userDefaults.string(forKey: mapStyleKey),
           let mapStyle = PinzMapStyle(rawValue: savedStyle) {
            return mapStyle
        }
        
        return .satelight
    }
}
