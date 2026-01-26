import _MapKit_SwiftUI

public enum PinzMapStyle: String, SegmentedItem {
    case scheme
    case hybrid
    case satelight

    public var id: Self { self }

    public var content: SegmentedItemContent {
        switch self {
        case .scheme: .text("Схема")
        case .satelight: .text("Спутник")
        case .hybrid: .text("Гибрид")
        }
    }

    public func toMapKitMapStyle() -> MapStyle {
        switch self {
        case .scheme: .standard
        case .satelight: .imagery
        case .hybrid: .hybrid
        }
    }
}
