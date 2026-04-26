public struct PinResponseDTO: Codable {
    public let pin: TripPinDTO

    public init(pin: TripPinDTO) {
        self.pin = pin
    }
}
