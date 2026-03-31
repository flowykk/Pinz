import SwiftUI

public struct RawPins: Hashable {
    public var pins: [RawPin]

    public init(pins: [RawPin]) {
        self.pins = pins
    }
}

public struct RawPin: Identifiable, Hashable {
    public let id: UUID
    public let serverDraftPinId: String?
    public var medias: [RawPinMedia]

    public init(medias: [RawPinMedia]) {
        self.id = UUID()
        self.serverDraftPinId = nil
        self.medias = medias
    }

    public init(serverDraftPinId: String, medias: [RawPinMedia]) {
        self.id = UUID()
        self.serverDraftPinId = serverDraftPinId
        self.medias = medias
    }
}

public struct RawPinMedia: Identifiable, Hashable {
    public let id: UUID
    public let serverMediaId: String?
    public var url: String
    public var type: MediaType

    public init(url: String, type: MediaType) {
        self.id = UUID()
        self.serverMediaId = nil
        self.url = url
        self.type = type
    }

    public init(serverMediaId: String, url: String, type: MediaType) {
        self.id = UUID()
        self.serverMediaId = serverMediaId
        self.url = url
        self.type = type
    }
}
