import SwiftUI

public struct RawPins {
    public var pins: [RawPin]

    public init(pins: [RawPin]) {
        self.pins = pins
    }
}

public struct RawPin {
    public var medias: [RawPinMedia]

    public init(medias: [RawPinMedia]) {
        self.medias = medias
    }
}

public struct RawPinMedia {
    public var url: String
    public var type: MediaType
}
