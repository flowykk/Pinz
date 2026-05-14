import Foundation
import Vapor
import PinzDomain

struct MockPinUploadStartRequest: Content, Equatable {
    let targetPinId: String?
    let filesToUpload: [MockFileToUploadRequest]

    enum CodingKeys: String, CodingKey {
        case targetPinId = "target_pin_id"
        case filesToUpload = "files_to_upload"
    }
}

struct MockFileToUploadRequest: Content, Equatable {
    let clientId: String
    let contentType: String

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case contentType = "content_type"
    }
}

struct MockPinUploadCommitRequest: Content, Equatable {
    let s3Key: String
    let mediaType: String
    let capturedAtUnix: Int?
    let latitude: Double?
    let longitude: Double?

    enum CodingKeys: String, CodingKey {
        case s3Key = "s3_key"
        case mediaType = "media_type"
        case capturedAtUnix = "captured_at_unix"
        case latitude
        case longitude
    }
}

struct MockPinUploadCommitResponse: Content {
    let mediaId: String
    let mediaCountInSession: Int

    enum CodingKeys: String, CodingKey {
        case mediaId = "media_id"
        case mediaCountInSession = "media_count_in_session"
    }
}

struct MockPinUploadFinalizeRequest: Content, Equatable {
    let name: String?
    let description: String?
    let category: String?
    let latitude: Double?
    let longitude: Double?
    let startTimeUnix: Int?
    let endTimeUnix: Int?
    let tags: [String]?
    let tagsSet: Bool?
    let mediaToDelete: [String]

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
        case mediaToDelete = "media_to_delete"
    }
}

actor MockPinUploadState {
    private let tripId: String
    private let sessionId: String
    private let createdPinId: String
    private let draftMediaId: String

    private var startCount = 0
    private var uploadCount = 0
    private var commitCount = 0
    private var processCount = 0
    private var reviewCount = 0
    private var finalizeCount = 0
    private var cancelCount = 0
    private var lastStartBody: MockPinUploadStartRequest?
    private var lastCommitBody: MockPinUploadCommitRequest?
    private var lastFinalizeBody: MockPinUploadFinalizeRequest?
    private var activeTargetPinId: String?

    init(
        tripId: String,
        sessionId: String,
        createdPinId: String,
        draftMediaId: String
    ) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.createdPinId = createdPinId
        self.draftMediaId = draftMediaId
    }

    func start(tripId: String, request: MockPinUploadStartRequest) -> Response {
        startCount += 1
        lastStartBody = request
        activeTargetPinId = request.targetPinId

        guard tripId == self.tripId else {
            return Response(status: .notFound)
        }

        let uploadURLs = request.filesToUpload.map {
            UploadURLDTO(
                clientId: $0.clientId,
                s3Key: "ui-tests/\($0.clientId).jpg",
                url: "http://localhost:8080/mock-upload/\($0.clientId)"
            )
        }
        let response = PinUploadStartResponseDTO(sessionId: sessionId, uploadUrls: uploadURLs)
        return encoded(response)
    }

    func upload() -> Response {
        uploadCount += 1
        return Response(status: .ok)
    }

    func commit(tripId: String, sessionId: String, request: MockPinUploadCommitRequest) -> Response {
        commitCount += 1
        lastCommitBody = request

        guard tripId == self.tripId, sessionId == self.sessionId else {
            return Response(status: .notFound)
        }

        return encoded(MockPinUploadCommitResponse(mediaId: draftMediaId, mediaCountInSession: commitCount))
    }

    func process(tripId: String, sessionId: String) -> Response {
        processCount += 1

        guard tripId == self.tripId, sessionId == self.sessionId else {
            return Response(status: .notFound)
        }

        return encoded(PinUploadProcessResponseDTO(sessionId: sessionId, processingStatus: "PROCESSING"))
    }

    func review(tripId: String, sessionId: String) -> Response {
        reviewCount += 1

        guard tripId == self.tripId, sessionId == self.sessionId else {
            return Response(status: .notFound)
        }

        let draft = PinUploadDraftDTO(
            suggested: PinSuggestedFieldsDTO(
                name: "Suggested UI Pin",
                category: "sight",
                tags: ["ui"],
                latitude: 55.7558,
                longitude: 37.6176,
                startTimeUnix: nil,
                endTimeUnix: nil
            ),
            media: [
                ReviewPinMediaDTO(
                    mediaId: draftMediaId,
                    url: "https://example.com/ui-pin.jpg",
                    privacyLevel: "private"
                )
            ],
            pinIssues: ["MISSING_DATES"],
            nsfwMediaIds: nil,
            dedupedMediaIds: nil
        )
        let response = PinUploadReviewResponseDTO(
            sessionId: sessionId,
            processingStatus: "READY_FOR_REVIEW",
            draft: draft,
            similar: nil
        )
        return encoded(response)
    }

    func finalize(tripId: String, sessionId: String, request: MockPinUploadFinalizeRequest) -> Response {
        finalizeCount += 1
        lastFinalizeBody = request

        guard tripId == self.tripId, sessionId == self.sessionId else {
            return Response(status: .notFound)
        }

        let pin = TripPinDTO(
            id: activeTargetPinId ?? createdPinId,
            tripId: tripId,
            name: request.name,
            description: request.description,
            category: request.category,
            latitude: request.latitude,
            longitude: request.longitude,
            startTimeUnix: request.startTimeUnix,
            endTimeUnix: request.endTimeUnix,
            tags: request.tags,
            privacyLevel: "private",
            media: [
                TripPinMediaDTO(
                    mediaId: draftMediaId,
                    url: "https://example.com/ui-pin.jpg",
                    mediaType: "image",
                    privacyLevel: "private",
                    capturedAtUnix: nil
                )
            ]
        )
        return encoded(PinResponseDTO(pin: pin))
    }

    func cancel(tripId: String, sessionId: String) -> Response {
        cancelCount += 1
        guard tripId == self.tripId, sessionId == self.sessionId else {
            return Response(status: .notFound)
        }
        return Response(status: .ok)
    }

    func counts() -> MockPinUploadCounts {
        MockPinUploadCounts(
            start: startCount,
            upload: uploadCount,
            commit: commitCount,
            process: processCount,
            review: reviewCount,
            finalize: finalizeCount,
            cancel: cancelCount
        )
    }

    func lastFinalizeRequest() -> MockPinUploadFinalizeRequest? {
        lastFinalizeBody
    }

    func lastStartRequest() -> MockPinUploadStartRequest? {
        lastStartBody
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

struct MockPinUploadCounts: Equatable {
    let start: Int
    let upload: Int
    let commit: Int
    let process: Int
    let review: Int
    let finalize: Int
    let cancel: Int
}

final class PinUploadResponseFactory: Sendable {
    private let state: MockPinUploadState

    init(
        tripId: String,
        sessionId: String = "pin-upload-ui-session-001",
        createdPinId: String = "pin-upload-ui-created-001",
        draftMediaId: String = "pin-upload-ui-media-001"
    ) {
        state = MockPinUploadState(
            tripId: tripId,
            sessionId: sessionId,
            createdPinId: createdPinId,
            draftMediaId: draftMediaId
        )
    }

    func start(tripId: String, request: MockPinUploadStartRequest) async -> Response {
        await state.start(tripId: tripId, request: request)
    }

    func upload() async -> Response {
        await state.upload()
    }

    func commit(tripId: String, sessionId: String, request: MockPinUploadCommitRequest) async -> Response {
        await state.commit(tripId: tripId, sessionId: sessionId, request: request)
    }

    func process(tripId: String, sessionId: String) async -> Response {
        await state.process(tripId: tripId, sessionId: sessionId)
    }

    func review(tripId: String, sessionId: String) async -> Response {
        await state.review(tripId: tripId, sessionId: sessionId)
    }

    func finalize(tripId: String, sessionId: String, request: MockPinUploadFinalizeRequest) async -> Response {
        await state.finalize(tripId: tripId, sessionId: sessionId, request: request)
    }

    func cancel(tripId: String, sessionId: String) async -> Response {
        await state.cancel(tripId: tripId, sessionId: sessionId)
    }

    func counts() async -> MockPinUploadCounts {
        await state.counts()
    }

    func lastFinalizeRequest() async -> MockPinUploadFinalizeRequest? {
        await state.lastFinalizeRequest()
    }

    func lastStartRequest() async -> MockPinUploadStartRequest? {
        await state.lastStartRequest()
    }
}
