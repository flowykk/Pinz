import Foundation
import SwiftUI

public struct Trip: Hashable {
    public var name: String
    public var image: UIImage?
    public var description: String?
    public var pins: [Pin]
    public var season: String?
    public var startDate: String?
    public var endDate: String?
    public var category: String?

    public init(
        name: String,
        image: UIImage? = nil,
        description: String? = nil,
        pins: [Pin],
        season: String? = nil,
        startDate: String? = nil,
        endDate: String? = nil,
        category: String? = nil
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
