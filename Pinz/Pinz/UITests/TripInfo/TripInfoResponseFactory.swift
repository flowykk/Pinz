import Foundation
import Vapor
import PinzDomain

extension TripDTO: Content {}
extension LeaveTripDTO: Content {}
extension DeletePinResponseDTO: Content {}

struct MockTripInfoSnapshot {
    let id: String
    var name: String
    var description: String?
    var category: String
    var season: String
    var privacyLevel: String?
    var coverUrl: String?
    var ownerUserId: String
    var startDateUnix: Int?
    var endDateUnix: Int?
    var createdAtUnix: Int
    var initialPins: [TripPinDTO] = []
}

struct MockGetTripResponse: Content {
    let trip: TripDTO
    let pins: [TripPinDTO]
    let participants: [TripParticipantDTO]
    let currentUserSettings: TripSettingsDTO?
    let activeAddMediaSession: ActiveAddMediaSessionDTO?

    enum CodingKeys: String, CodingKey {
        case trip
        case pins
        case participants
        case currentUserSettings = "current_user_settings"
        case activeAddMediaSession = "active_add_media_session"
    }
}

struct MockTripUpdateErrorResponse: Content {
    let error: String
}

struct MockUpdateTripRequest: Content, Equatable {
    let name: String?
    let description: String?
    let category: String?
    let season: String?
    let privacyLevel: String?
    let coverUrl: String?
    let startDateUnix: Int?
    let endDateUnix: Int?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case category
        case season
        case privacyLevel = "privacy_level"
        case coverUrl = "cover_url"
        case startDateUnix = "start_date_unix"
        case endDateUnix = "end_date_unix"
    }
}

struct MockUpdatePinRequest: Content, Equatable {
    let name: String?
    let description: String?
    let category: String?
    let latitude: Double?
    let longitude: Double?
    let startTimeUnix: Int?
    let endTimeUnix: Int?
    let tags: [String]?
    let tagsSet: Bool?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case category
        case latitude
        case longitude
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
        case tags
        case tagsSet = "tags_set"
    }
}

struct MockUpdatePinResponse: Content {
    let pin: TripPinDTO
}

actor MockTripInfoState {
    private var snapshot: MockTripInfoSnapshot
    private let expectedTripId: String
    private let patchShouldFail: Bool
    private let patchFailureStatus: HTTPResponseStatus
    private let patchFailureMessage: String
    private let patchPinShouldFail: Bool
    private let patchPinFailureStatus: HTTPResponseStatus
    private let patchPinFailureMessage: String

    private var getTripCount = 0
    private var patchTripCount = 0
    private var patchPinCount = 0
    private var deletePinCount = 0
    private var deleteTripCount = 0
    private var leaveTripCount = 0
    private var lastPatchBody: MockUpdateTripRequest?
    private var lastPinPatchBody: MockUpdatePinRequest?
    private var lastDeletedPinId: String?

    init(
        initialTrip: MockTripInfoSnapshot,
        expectedTripId: String,
        patchShouldFail: Bool,
        patchFailureStatus: HTTPResponseStatus,
        patchFailureMessage: String,
        patchPinShouldFail: Bool,
        patchPinFailureStatus: HTTPResponseStatus,
        patchPinFailureMessage: String
    ) {
        self.snapshot = initialTrip
        self.expectedTripId = expectedTripId
        self.patchShouldFail = patchShouldFail
        self.patchFailureStatus = patchFailureStatus
        self.patchFailureMessage = patchFailureMessage
        self.patchPinShouldFail = patchPinShouldFail
        self.patchPinFailureStatus = patchPinFailureStatus
        self.patchPinFailureMessage = patchPinFailureMessage
    }

    func getTrip(tripId: String) async -> Response {
        getTripCount += 1
        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        return tripResponse()
    }

    func patchTrip(tripId: String, request: MockUpdateTripRequest) async -> Response {
        patchTripCount += 1
        lastPatchBody = request

        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        if let validationErrorResponse = validatePatchRequest(request) {
            return validationErrorResponse
        }

        if patchShouldFail {
            let response = Response(status: patchFailureStatus)
            try? response.content.encode(MockTripUpdateErrorResponse(error: patchFailureMessage))
            return response
        }

        if let name = request.name {
            snapshot.name = name
        }
        if let description = request.description {
            snapshot.description = description
        }
        if let category = request.category {
            snapshot.category = category
        }
        if let season = request.season {
            snapshot.season = season
        }
        if let privacyLevel = request.privacyLevel {
            snapshot.privacyLevel = privacyLevel
        }
        if let coverUrl = request.coverUrl {
            snapshot.coverUrl = coverUrl
        }
        if let startDateUnix = request.startDateUnix {
            snapshot.startDateUnix = startDateUnix
        }
        if let endDateUnix = request.endDateUnix {
            snapshot.endDateUnix = endDateUnix
        }

        return tripPatchResponse()
    }

    func patchPin(tripId: String, pinId: String, request: MockUpdatePinRequest) async -> Response {
        patchPinCount += 1
        lastPinPatchBody = request

        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        guard let idx = snapshot.initialPins.firstIndex(where: { $0.id == pinId }) else {
            return Response(status: .notFound)
        }

        if let validationErrorResponse = validatePatchPinRequest(request) {
            return validationErrorResponse
        }

        if patchPinShouldFail {
            let response = Response(status: patchPinFailureStatus)
            try? response.content.encode(MockTripUpdateErrorResponse(error: patchPinFailureMessage))
            return response
        }

        snapshot.initialPins[idx] = applyPatch(to: snapshot.initialPins[idx], with: request)
        return pinPatchResponse(pin: snapshot.initialPins[idx])
    }

    func deleteTrip(tripId: String) async -> Response {
        deleteTripCount += 1

        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        return tripDeleteResponse()
    }

    func deletePin(tripId: String, pinId: String) async -> Response {
        deletePinCount += 1
        lastDeletedPinId = pinId

        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        guard let idx = snapshot.initialPins.firstIndex(where: { $0.id == pinId }) else {
            return Response(status: .notFound)
        }

        snapshot.initialPins.remove(at: idx)
        let response = Response(status: .ok)
        try? response.content.encode(DeletePinResponseDTO(deletionMode: "full"))
        return response
    }

    func leaveTrip(tripId: String) async -> Response {
        leaveTripCount += 1

        guard tripIdMatchesExpected(tripId: tripId) else {
            return Response(status: .notFound)
        }

        return tripLeaveResponse()
    }

    func getTripCount() async -> Int {
        getTripCount
    }

    func patchTripCount() async -> Int {
        patchTripCount
    }

    func patchPinCount() async -> Int {
        patchPinCount
    }

    func deletePinCount() async -> Int {
        deletePinCount
    }

    func deleteTripCount() async -> Int {
        deleteTripCount
    }

    func leaveTripCount() async -> Int {
        leaveTripCount
    }

    func lastPatchBody() async -> MockUpdateTripRequest? {
        lastPatchBody
    }

    func lastPinPatchBody() async -> MockUpdatePinRequest? {
        lastPinPatchBody
    }

    func lastDeletedPinId() async -> String? {
        lastDeletedPinId
    }

    private func tripIdMatchesExpected(tripId: String) -> Bool {
        tripId == expectedTripId
    }

    private func validatePatchRequest(_ request: MockUpdateTripRequest) -> Response? {
        if request.name == nil && request.description == nil && request.category == nil
            && request.season == nil && request.privacyLevel == nil && request.coverUrl == nil
            && request.startDateUnix == nil && request.endDateUnix == nil {
            let response = Response(status: .badRequest)
            try? response.content.encode(MockTripUpdateErrorResponse(error: "empty_update_payload"))
            return response
        }
        return nil
    }

    private func validatePatchPinRequest(_ request: MockUpdatePinRequest) -> Response? {
        if request.name == nil && request.description == nil && request.category == nil
            && request.latitude == nil && request.longitude == nil
            && request.startTimeUnix == nil && request.endTimeUnix == nil
            && request.tags == nil && request.tagsSet == nil {
            let response = Response(status: .badRequest)
            try? response.content.encode(MockTripUpdateErrorResponse(error: "empty_update_payload"))
            return response
        }
        return nil
    }

    private func applyPatch(to pin: TripPinDTO, with request: MockUpdatePinRequest) -> TripPinDTO {
        let hasTagsUpdate = request.tagsSet == true

        return TripPinDTO(
            id: pin.id,
            tripId: pin.tripId,
            name: request.name ?? pin.name,
            description: request.description ?? pin.description,
            category: request.category ?? pin.category,
            latitude: request.latitude ?? pin.latitude,
            longitude: request.longitude ?? pin.longitude,
            startTimeUnix: request.startTimeUnix ?? pin.startTimeUnix,
            endTimeUnix: request.endTimeUnix ?? pin.endTimeUnix,
            tags: hasTagsUpdate ? request.tags : pin.tags,
            privacyLevel: pin.privacyLevel,
            media: pin.media
        )
    }

    private func tripResponse() -> Response {
        let trip = tripDTO()

        let responseBody = MockGetTripResponse(
            trip: trip,
            pins: snapshot.initialPins,
            participants: [],
            currentUserSettings: nil,
            activeAddMediaSession: nil
        )

        let response = Response(status: .ok)
        try? response.content.encode(responseBody)
        return response
    }

    private func tripPatchResponse() -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(tripDTO())
        return response
    }

    private func pinPatchResponse(pin: TripPinDTO) -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(MockUpdatePinResponse(pin: pin))
        return response
    }

    private func tripDeleteResponse() -> Response {
        Response(status: .ok)
    }

    private func tripLeaveResponse() -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(LeaveTripDTO(success: true, tripDeleted: false))
        return response
    }

    private func tripDTO() -> TripDTO {
        var dto = snapshot

        // Keep createdAt stable for easier assertions and prevent changing payload drift.
        if dto.createdAtUnix == 0 {
            dto.createdAtUnix = Int(Date().timeIntervalSince1970)
        }

        return TripDTO(
            id: dto.id,
            name: dto.name,
            description: dto.description,
            category: dto.category,
            season: dto.season,
            coverUrl: dto.coverUrl,
            ownerUserId: dto.ownerUserId,
            privacyLevel: dto.privacyLevel,
            status: nil,
            isPublished: false,
            isGenerated: false,
            likesCount: 0,
            dislikesCount: 0,
            participantsCount: 0,
            mediaCount: 0,
            startDateUnix: dto.startDateUnix,
            endDateUnix: dto.endDateUnix,
            createdAtUnix: dto.createdAtUnix,
            updatedAtUnix: Int(Date().timeIntervalSince1970)
        )
    }
}

struct TripInfoResponseFactory {
    private let state: MockTripInfoState

    init(
        initialTrip: MockTripInfoSnapshot,
        patchShouldFail: Bool = false,
        patchFailureStatus: HTTPResponseStatus = .internalServerError,
        patchFailureMessage: String = "Failed to update trip",
        patchPinShouldFail: Bool = false,
        patchPinFailureStatus: HTTPResponseStatus = .internalServerError,
        patchPinFailureMessage: String = "Failed to update pin"
    ) {
        state = MockTripInfoState(
            initialTrip: initialTrip,
            expectedTripId: initialTrip.id,
            patchShouldFail: patchShouldFail,
            patchFailureStatus: patchFailureStatus,
            patchFailureMessage: patchFailureMessage,
            patchPinShouldFail: patchPinShouldFail,
            patchPinFailureStatus: patchPinFailureStatus,
            patchPinFailureMessage: patchPinFailureMessage
        )
    }

    func getTrip(tripId: String) async -> Response {
        await state.getTrip(tripId: tripId)
    }

    func patchTrip(tripId: String, request: MockUpdateTripRequest) async -> Response {
        await state.patchTrip(tripId: tripId, request: request)
    }

    func patchPin(tripId: String, pinId: String, request: MockUpdatePinRequest) async -> Response {
        await state.patchPin(tripId: tripId, pinId: pinId, request: request)
    }

    func deletePin(tripId: String, pinId: String) async -> Response {
        await state.deletePin(tripId: tripId, pinId: pinId)
    }

    func deleteTrip(tripId: String) async -> Response {
        await state.deleteTrip(tripId: tripId)
    }

    func leaveTrip(tripId: String) async -> Response {
        await state.leaveTrip(tripId: tripId)
    }

    func getTripCount() async -> Int {
        await state.getTripCount()
    }

    func patchTripCount() async -> Int {
        await state.patchTripCount()
    }

    func patchPinCount() async -> Int {
        await state.patchPinCount()
    }

    func deletePinCount() async -> Int {
        await state.deletePinCount()
    }

    func deleteTripCount() async -> Int {
        await state.deleteTripCount()
    }

    func leaveTripCount() async -> Int {
        await state.leaveTripCount()
    }

    func lastPatchBody() async -> MockUpdateTripRequest? {
        await state.lastPatchBody()
    }

    func lastPinPatchBody() async -> MockUpdatePinRequest? {
        await state.lastPinPatchBody()
    }

    func lastDeletedPinId() async -> String? {
        await state.lastDeletedPinId()
    }
}
