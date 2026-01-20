import Foundation
import SwiftUI

public struct Trip: Hashable {
    public var name: String
    public var image: UIImage?
    public var description: String
    public var pins: [Pin]
    public var season: TripSeason
    public var startDate: String?
    public var endDate: String?
    public var category: TripCategory

    public init(
        name: String,
        image: UIImage? = nil,
        description: String = "",
        pins: [Pin],
        season: TripSeason,
        startDate: String? = nil,
        endDate: String? = nil,
        category: TripCategory
    ) {
        self.name = name
        self.image = image
        self.description = description
        self.pins = pins
        self.season = season
        self.startDate = startDate
        self.endDate = endDate
        self.category = category
    }
}
