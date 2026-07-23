public struct PinResponseDTO: Codable {
    public let pin: TripPinDTO

    public init(pin: TripPinDTO) {
        self.pin = pin
    }
}

public extension PinResponseDTO {
    func toPin(tripId: String? = nil, nameIfMissing: String = "") -> Pin {
        pin.toPin(index: 0, tripId: tripId ?? pin.tripId, nameIfMissing: nameIfMissing)
    }
}
