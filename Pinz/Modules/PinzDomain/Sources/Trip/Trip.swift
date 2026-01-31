import Foundation
import SwiftUI

public struct Trip: Hashable {
    public var name: String
    public var image: UIImage?
    public var description: String?
    public var pins: [Pin]
    public var season: TripSeason
    public var startDate: Date?
    public var endDate: Date?
    public var category: TripCategory
    public var members: [TripMember]

    public init(
        name: String,
        image: UIImage? = nil,
        description: String? = nil,
        pins: [Pin],
        season: TripSeason,
        startDate: Date? = nil,
        endDate: Date? = nil,
        category: TripCategory,
        members: [TripMember] = []
    ) {
        self.name = name
        self.image = image
        self.description = description
        self.pins = pins
        self.season = season
        self.startDate = startDate
        self.endDate = endDate
        self.category = category
        self.members = members
    }
}
