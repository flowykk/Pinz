import Foundation
import CoreLocation

public struct PlaceSaveAction: Equatable, Hashable {
    public let id = UUID()
    public let action: (CLLocationCoordinate2D?) -> Void
    
    public init(action: @escaping (CLLocationCoordinate2D?) -> Void) {
        self.action = action
    }
    
    public static func == (lhs: PlaceSaveAction, rhs: PlaceSaveAction) -> Bool {
        lhs.id == rhs.id
    }
    
    public func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}
