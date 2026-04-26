import Foundation

public struct GetTripResponseDTO: Codable {
    public let trip: TripDTO
    public let pins: [TripPinDTO]
    public let participants: [TripParticipantDTO]
    public let currentUserSettings: TripSettingsDTO?
    public let activeAddMediaSession: ActiveAddMediaSessionDTO?

    enum CodingKeys: String, CodingKey {
        case trip, pins, participants
        case currentUserSettings = "current_user_settings"
        case activeAddMediaSession = "active_add_media_session"
    }

    public init(
        trip: TripDTO,
        pins: [TripPinDTO],
        participants: [TripParticipantDTO] = [],
        currentUserSettings: TripSettingsDTO? = nil,
        activeAddMediaSession: ActiveAddMediaSessionDTO? = nil
    ) {
        self.trip = trip
        self.pins = pins
        self.participants = participants
        self.currentUserSettings = currentUserSettings
        self.activeAddMediaSession = activeAddMediaSession
    }
}
