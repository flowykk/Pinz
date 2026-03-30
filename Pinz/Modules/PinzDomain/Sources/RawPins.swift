import SwiftUI

public struct RawPins {
    public var pins: [RawPin]

    public init(pins: [RawPin]) {
        self.pins = pins
    }
}

public struct RawPin: Identifiable {
    public let id: UUID
    public var medias: [RawPinMedia]

    public init(medias: [RawPinMedia]) {
        self.id = UUID()
        self.medias = medias
    }
}

public struct RawPinMedia: Identifiable {
    public let id: UUID
    public var url: String
    public var type: MediaType

    public init(url: String, type: MediaType) {
        self.id = UUID()
        self.url = url
        self.type = type
    }
}
