import Foundation
import Vapor
import PinzDomain

struct MockCreateTripRequest: Content, Equatable {
    let name: String
    let description: String?
    let category: String?
    let season: String?
    let filesToUpload: [MockFileToUploadRequest]

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case category
        case season
        case filesToUpload = "files_to_upload"
    }
}

struct MockMediaMetaEntryRequest: Content, Equatable {
    let s3Key: String
    let capturedAt: String?
    let latitude: Double?
    let longitude: Double?
    let mediaType: String
    let contentHash: String?

    enum CodingKeys: String, CodingKey {
        case s3Key = "s3_key"
        case capturedAt = "captured_at"
        case latitude
        case longitude
        case mediaType = "media_type"
        case contentHash = "content_hash"
    }
}

struct MockProcessMediaGroupingRequest: Content, Equatable {
    let media: [MockMediaMetaEntryRequest]
}

struct MockDraftPinInputRequest: Content, Equatable {
    let draftPinId: String
    let mediaIds: [String]

    enum CodingKeys: String, CodingKey {
        case draftPinId = "draft_pin_id"
        case mediaIds = "media_ids"
    }
}

struct MockApplyGroupsAndProcessRequest: Content, Equatable {
    let draftPins: [MockDraftPinInputRequest]
    let deletedMediaIds: [String]

    enum CodingKeys: String, CodingKey {
        case draftPins = "draft_pins"
        case deletedMediaIds = "deleted_media_ids"
    }
}

struct MockFinalizeTripRequest: Content, Equatable {
    let pinUpdates: [MockPinUpdateInputRequest]
    let mediaToDelete: [String]

    enum CodingKeys: String, CodingKey {
        case pinUpdates = "pin_updates"
        case mediaToDelete = "media_to_delete"
    }
}

struct MockPinUpdateInputRequest: Content, Equatable {
    let pinId: String
    let name: String?
    let description: String?
    let category: String?
    let privacyLevel: String?
    let latitude: Double?
    let longitude: Double?
    let tags: [String]?
    let startTimeUnix: Int?
    let endTimeUnix: Int?

    enum CodingKeys: String, CodingKey {
        case pinId = "pin_id"
        case name
        case description
        case category
        case privacyLevel = "privacy_level"
        case latitude
        case longitude
        case tags
        case startTimeUnix = "start_time_unix"
        case endTimeUnix = "end_time_unix"
    }
}

actor MockTripCreationState {
    private let tripId: String
    private let draftPinId: String
    private let mediaId: String

    private var createCount = 0
    private var uploadCount = 0
    private var processGroupingCount = 0
    private var applyGroupsCount = 0
    private var reviewCount = 0
    private var finalizeCount = 0
    private var lastCreateBody: MockCreateTripRequest?
    private var lastProcessGroupingBody: MockProcessMediaGroupingRequest?
    private var lastApplyGroupsBody: MockApplyGroupsAndProcessRequest?
    private var lastFinalizeBody: MockFinalizeTripRequest?

    init(
        tripId: String = "trip-creation-ui-001",
        draftPinId: String = "draft-pin-ui-001",
        mediaId: String = "trip-creation-media-ui-001"
    ) {
        self.tripId = tripId
        self.draftPinId = draftPinId
        self.mediaId = mediaId
    }

    func create(request: MockCreateTripRequest) -> Response {
        createCount += 1
        lastCreateBody = request

        let uploadURLs = request.filesToUpload.map {
            UploadURLDTO(
                clientId: $0.clientId,
                s3Key: "ui-tests/trip-creation/\($0.clientId).jpg",
                url: "http://localhost:8080/mock-trip-creation-upload/\($0.clientId)"
            )
        }
        return encoded(CreateTripDTO(tripId: tripId, status: "created", uploadUrls: uploadURLs))
    }

    func upload() -> Response {
        uploadCount += 1
        return Response(status: .ok)
    }

    func processGrouping(tripId: String, request: MockProcessMediaGroupingRequest) -> Response {
        processGroupingCount += 1
        lastProcessGroupingBody = request

        guard tripId == self.tripId else {
            return Response(status: .notFound)
        }

        let media = DraftPinMediaDTO(
            mediaId: mediaId,
            type: "image",
            url: "https://example.com/trip-creation-media.jpg"
        )
        let response = ProcessMediaGroupingDTO(
            tripId: tripId,
            status: "processed",
            draftPins: [DraftPinDTO(draftPinId: draftPinId, media: [media])]
        )
        return encoded(response)
    }

    func applyGroupsAndProcess(tripId: String, request: MockApplyGroupsAndProcessRequest) -> Response {
        applyGroupsCount += 1
        lastApplyGroupsBody = request

        guard tripId == self.tripId else {
            return Response(status: .notFound)
        }

        return encoded(ApplyGroupsAndProcessDTO(message: "processing", status: "ok"))
    }

    func review(tripId: String) -> Response {
        reviewCount += 1

        guard tripId == self.tripId else {
            return Response(status: .notFound)
        }

        let media = ReviewPinMediaDTO(
            mediaId: mediaId,
            url: "https://example.com/trip-creation-media.jpg",
            privacyLevel: "private"
        )
        let pin = ReviewPinDTO(
            pinId: "review-pin-ui-001",
            name: "Created Trip Review Pin",
            category: "sight",
            latitude: 55.7558,
            longitude: 37.6176,
            tags: ["ui"],
            issues: [],
            media: [media]
        )
        return encoded(GetTripReviewDTO(tripId: tripId, status: "DRAFT_FINAL_REVIEW", pins: [pin], similar: []))
    }

    func finalize(tripId: String, request: MockFinalizeTripRequest) -> Response {
        finalizeCount += 1
        lastFinalizeBody = request

        guard tripId == self.tripId else {
            return Response(status: .notFound)
        }

        return encoded(FinalizeTripDTO(tripId: tripId, status: "finalized", message: "done"))
    }

    func counts() -> MockTripCreationCounts {
        MockTripCreationCounts(
            create: createCount,
            upload: uploadCount,
            processGrouping: processGroupingCount,
            applyGroups: applyGroupsCount,
            review: reviewCount,
            finalize: finalizeCount
        )
    }

    func lastCreateRequest() -> MockCreateTripRequest? {
        lastCreateBody
    }

    func lastApplyGroupsRequest() -> MockApplyGroupsAndProcessRequest? {
        lastApplyGroupsBody
    }

    func lastFinalizeRequest() -> MockFinalizeTripRequest? {
        lastFinalizeBody
    }

    private func encoded<T: Encodable>(_ value: T, status: HTTPResponseStatus = .ok) -> Response {
        let response = Response(status: status)
        response.headers.contentType = .json
        if let data = try? JSONEncoder().encode(value) {
            response.body = .init(data: data)
        }
        return response
    }
}

struct MockTripCreationCounts: Equatable {
    let create: Int
    let upload: Int
    let processGrouping: Int
    let applyGroups: Int
    let review: Int
    let finalize: Int
}

final class TripCreationResponseFactory: Sendable {
    private let state: MockTripCreationState

    init(tripId: String = "trip-creation-ui-001") {
        state = MockTripCreationState(tripId: tripId)
    }

    func create(request: MockCreateTripRequest) async -> Response {
        await state.create(request: request)
    }

    func upload() async -> Response {
        await state.upload()
    }

    func processGrouping(tripId: String, request: MockProcessMediaGroupingRequest) async -> Response {
        await state.processGrouping(tripId: tripId, request: request)
    }

    func applyGroupsAndProcess(tripId: String, request: MockApplyGroupsAndProcessRequest) async -> Response {
        await state.applyGroupsAndProcess(tripId: tripId, request: request)
    }

    func review(tripId: String) async -> Response {
        await state.review(tripId: tripId)
    }

    func finalize(tripId: String, request: MockFinalizeTripRequest) async -> Response {
        await state.finalize(tripId: tripId, request: request)
    }

    func counts() async -> MockTripCreationCounts {
        await state.counts()
    }

    func lastCreateRequest() async -> MockCreateTripRequest? {
        await state.lastCreateRequest()
    }

    func lastApplyGroupsRequest() async -> MockApplyGroupsAndProcessRequest? {
        await state.lastApplyGroupsRequest()
    }

    func lastFinalizeRequest() async -> MockFinalizeTripRequest? {
        await state.lastFinalizeRequest()
    }
}
