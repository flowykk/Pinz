import SwiftUI
import PinzBase
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
        case .scheme: .text(PinzBaseStrings.MapStyle.Label.standard)
        case .satelight: .text(PinzBaseStrings.MapStyle.Label.satellite)
        case .hybrid: .text(PinzBaseStrings.MapStyle.Label.hybrid)
        }
    }

    public func toMapKitMapStyle() -> MapStyle {
        switch self {
        case .scheme: .standard
        case .satelight: .imagery
        case .hybrid: .hybrid
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
