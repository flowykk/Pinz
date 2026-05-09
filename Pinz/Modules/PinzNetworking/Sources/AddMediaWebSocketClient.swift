import Foundation
import PinzBase

// MARK: - Public event type

public enum AddMediaWSEvent {
    case tripStatusChanged(status: String)
    case addMediaProgress(
        mediaId: String,
        mediaUrl: String,
        mediaType: String,
        actorUserId: String,
        mediaCount: Int
    )
    case initiatorChanged(
        previousInitiatorId: String?,
        currentInitiatorId: String,
        takeoverAvailableAtUnix: Int
    )
    case unknown
}

// MARK: - Client

public final class AddMediaWebSocketClient {

    private var receiveTask: Task<Void, Never>?
    private var webSocketTask: URLSessionWebSocketTask?
    private let session: URLSession

    public init(session: URLSession = .shared) {
        self.session = session
    }

    // MARK: - AsyncStream API (for ViewModels)

    public func connect(tripId: String) -> AsyncStream<AddMediaWSEvent> {
        disconnect()

        let wsTask = makeWebSocketTask(for: tripId)
        webSocketTask = wsTask
        wsTask.resume()

        return AsyncStream { [weak self] continuation in
            guard let self else {
                continuation.finish()
                return
            }

            let task = Task {
                await self.receiveLoop(wsTask: wsTask, continuation: continuation)
            }

            receiveTask = task

            continuation.onTermination = { [weak self] _ in
                task.cancel()
                self?.webSocketTask?.cancel(with: .normalClosure, reason: nil)
            }
        }
    }

    public func disconnect() {
        receiveTask?.cancel()
        receiveTask = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
    }

    // MARK: - Blocking wait API (for NetworkService)

    func waitForDraftFinalReview(tripId: String, timeout: TimeInterval) async throws {
        let wsTask = makeWebSocketTask(for: tripId)
        wsTask.resume()

        defer {
            wsTask.cancel(with: .normalClosure, reason: nil)
        }

        try await withThrowingTaskGroup(of: Void.self) { group in
            group.addTask {
                try await Self.waitForStatusEvent(
                    wsTask: wsTask,
                    expectedStatus: "ADD_MEDIA_DRAFT_FINAL_REVIEW"
                )
            }

            group.addTask {
                if timeout <= 0 { throw TripReviewWaitError.timeout }
                do {
                    try await Task.sleep(nanoseconds: UInt64(timeout * 1_000_000_000))
                } catch {
                    throw TripReviewWaitError.cancelled
                }
                throw TripReviewWaitError.timeout
            }

            do {
                try await group.next()
                group.cancelAll()
            } catch {
                group.cancelAll()
                throw error
            }
        }
    }

    // MARK: - Private helpers

    private func receiveLoop(
        wsTask: URLSessionWebSocketTask,
        continuation: AsyncStream<AddMediaWSEvent>.Continuation
    ) async {
        defer { continuation.finish() }

        while !Task.isCancelled {
            do {
                let message = try await wsTask.receive()
                let data = Self.data(from: message)
                let event = Self.parseEvent(from: data)
                continuation.yield(event)
            } catch {
                break
            }
        }
    }

    private static func waitForStatusEvent(
        wsTask: URLSessionWebSocketTask,
        expectedStatus: String
    ) async throws {
        let target = normalize(expectedStatus)

        while !Task.isCancelled {
            do {
                let message = try await wsTask.receive()
                let data = Self.data(from: message)
                let event = Self.parseEvent(from: data)

                if case .tripStatusChanged(let status) = event, normalize(status) == target {
                    return
                }
            } catch {
                if error is CancellationError || Task.isCancelled {
                    throw TripReviewWaitError.cancelled
                }
                throw TripReviewWaitError.webSocket(message: error.localizedDescription)
            }
        }

        throw TripReviewWaitError.cancelled
    }

    private func makeWebSocketTask(for tripId: String) -> URLSessionWebSocketTask {
        let url = websocketURL(for: tripId)
        var request = URLRequest(url: url)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = TokenStorage.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return session.webSocketTask(with: request)
    }

    private func websocketURL(for tripId: String) -> URL {
        let base = commandLineHostURL()
        var components = URLComponents()
        components.scheme = base.scheme
        components.host = base.host
        components.port = base.port
        components.path = "/api/v1/trips/\(tripId)/events/ws"
        return components.url ?? base
    }

    private func commandLineHostURL() -> URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "ws://localhost:8080")!
        }
        return URL(string: "wss://pinz.website")!
    }

    private static func data(from message: URLSessionWebSocketTask.Message) -> Data {
        switch message {
        case .string(let string): return Data(string.utf8)
        case .data(let bytes): return bytes
        @unknown default: return Data()
        }
    }

    private static func normalize(_ raw: String) -> String {
        raw.uppercased()
            .replacingOccurrences(of: "_", with: "")
            .replacingOccurrences(of: " ", with: "")
    }
}

// MARK: - Event parsing

extension AddMediaWebSocketClient {

    static func parseEvent(from data: Data) -> AddMediaWSEvent {
        guard let envelope = try? JSONDecoder().decode(WSEnvelope.self, from: data) else {
            return .unknown
        }

        let eventType = normalize(envelope.resolvedType)

        switch eventType {
        case normalize("TRIP_STATUS_CHANGED"):
            if let p = try? JSONDecoder().decode(WSWithPayload<TripStatusPayload>.self, from: data) {
                return .tripStatusChanged(status: p.payload.status)
            }

        case normalize("ADD_MEDIA_PROGRESS"):
            if let p = try? JSONDecoder().decode(WSWithPayload<AddMediaProgressPayload>.self, from: data) {
                return .addMediaProgress(
                    mediaId: p.payload.media.mediaId,
                    mediaUrl: p.payload.media.url,
                    mediaType: p.payload.media.mediaType,
                    actorUserId: p.payload.actorUserId,
                    mediaCount: p.payload.mediaCount
                )
            }

        case normalize("ADD_MEDIA_INITIATOR_CHANGED"):
            if let p = try? JSONDecoder().decode(WSWithPayload<InitiatorChangedPayload>.self, from: data) {
                return .initiatorChanged(
                    previousInitiatorId: p.payload.previousInitiatorUserId,
                    currentInitiatorId: p.payload.currentInitiatorUserId,
                    takeoverAvailableAtUnix: p.payload.takeoverAvailableAtUnix
                )
            }

        default:
            break
        }

        return .unknown
    }
}

// MARK: - Private Decodable types

private struct WSEnvelope: Decodable {
    let type: String?
    let event: String?

    var resolvedType: String { type ?? event ?? "" }
}

private struct WSWithPayload<P: Decodable>: Decodable {
    let payload: P
}

private struct TripStatusPayload: Decodable {
    let status: String
}

private struct AddMediaProgressPayload: Decodable {
    let media: ProgressMedia
    let actorUserId: String
    let mediaCount: Int

    struct ProgressMedia: Decodable {
        let mediaId: String
        let mediaType: String
        let url: String

        enum CodingKeys: String, CodingKey {
            case mediaId = "media_id"
            case mediaType = "media_type"
            case url
        }
    }

    enum CodingKeys: String, CodingKey {
        case media
        case actorUserId = "actor_user_id"
        case mediaCount = "media_count"
    }
}

private struct InitiatorChangedPayload: Decodable {
    let previousInitiatorUserId: String?
    let currentInitiatorUserId: String
    let takeoverAvailableAtUnix: Int

    enum CodingKeys: String, CodingKey {
        case previousInitiatorUserId = "previous_initiator_user_id"
        case currentInitiatorUserId = "current_initiator_user_id"
        case takeoverAvailableAtUnix = "takeover_available_at_unix"
    }
}
