import SwiftUI

public struct Trip: Hashable {
    public var name: String
    public var image: UIImage?
    public var pins: [Pin]

    public init(name: String, image: UIImage? = nil, pins: [Pin]) {
        self.name = name
        self.image = image
        self.pins = pins
    }
}
