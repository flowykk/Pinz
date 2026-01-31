import Foundation
import CoreLocation

public struct Pin: Hashable, Identifiable {
    public var id: String { name }
    
    public var name: String
    public var description: String
    public var category: PinCategory
    public var medias: [LoadedMedia]
    public var privacy: Bool
    public var startDate: Date?
    public var endDate: Date?
    public var tags: [MediaTag]
    public var coordinates: CLLocationCoordinate2D

    public init(
        name: String,
        description: String,
        category: PinCategory,
        medias: [LoadedMedia],
        privacy: Bool,
        startDate: Date? = nil,
        endDate: Date? = nil,
        tags: [MediaTag],
        coordinates: CLLocationCoordinate2D
    ) {
        self.name = name
        self.description = description
        self.category = category
        self.medias = medias
        self.privacy = privacy
        self.startDate = startDate
        self.endDate = endDate
        self.tags = tags
        self.coordinates = coordinates
    }
}

extension CLLocationCoordinate2D: Hashable {
    public func hash(into hasher: inout Hasher) {
        hasher.combine(latitude)
        hasher.combine(longitude)
    }
    
    public static func == (lhs: CLLocationCoordinate2D, rhs: CLLocationCoordinate2D) -> Bool {
        lhs.latitude == rhs.latitude && lhs.longitude == rhs.longitude
    }
}
