import SwiftUI

public struct RawPins: Hashable {
    public var pins: [RawPin]

    public init(pins: [RawPin]) {
        self.pins = pins
    }
}

public struct RawPin: Identifiable, Hashable {
    public let id: String
    public var medias: [RawPinMedia]

    public init(id: String, medias: [RawPinMedia]) {
        self.id = id
        self.medias = medias
    }
}

public struct RawPinMedia: Identifiable, Hashable {
    public let id: String
    public var url: String
    public var type: MediaType

    public init(id: String, url: String, type: MediaType) {
        self.id = id
        self.url = url
        self.type = type
    }
}
