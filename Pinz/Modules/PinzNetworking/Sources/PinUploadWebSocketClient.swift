import Foundation
import PinzBase

// MARK: - Public event type

public enum PinUploadWSEvent {
    case processingCompleted(sessionId: String, targetPinId: String?, processingStatus: String)
    case unknown
}

// MARK: - Client

public final class PinUploadWebSocketClient {

    private var receiveTask: Task<Void, Never>?
    private var webSocketTask: URLSessionWebSocketTask?
    private let session: URLSession

    public init(session: URLSession = .shared) {
        self.session = session
    }

    // MARK: - AsyncStream API

    public func connect(tripId: String, sessionId: String) -> AsyncStream<PinUploadWSEvent> {
        disconnect()

        let url = websocketURL(tripId: tripId, sessionId: sessionId)
        let hasToken = TokenStorage.shared.accessToken != nil
        print("[PinUploadWS] connect tripId=\(tripId) sessionId=\(sessionId) url=\(url.absoluteString) hasToken=\(hasToken)")

        let wsTask = makeWebSocketTask(url: url)
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
                print("[PinUploadWS] stream terminated sessionId=\(sessionId)")
                task.cancel()
                self?.webSocketTask?.cancel(with: .normalClosure, reason: nil)
            }
        }
    }

    public func disconnect() {
        guard receiveTask != nil || webSocketTask != nil else { return }
        print("[PinUploadWS] disconnect")
        receiveTask?.cancel()
        receiveTask = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
    }

    // MARK: - Private helpers

    private func receiveLoop(
        wsTask: URLSessionWebSocketTask,
        continuation: AsyncStream<PinUploadWSEvent>.Continuation
    ) async {
        defer {
            print("[PinUploadWS] receive loop ended")
            continuation.finish()
        }

        print("[PinUploadWS] receive loop started")

        while !Task.isCancelled {
            do {
                let message = try await wsTask.receive()
                let data = Self.data(from: message)
                if let raw = String(data: data, encoding: .utf8) {
                    print("[PinUploadWS] ← message: \(raw)")
                } else {
                    print("[PinUploadWS] ← message: <\(data.count) bytes binary>")
                }
                let event = Self.parseEvent(from: data)
                print("[PinUploadWS] parsed event: \(Self.describe(event))")
                continuation.yield(event)
            } catch {
                if Task.isCancelled {
                    print("[PinUploadWS] receive cancelled")
                } else {
                    print("[PinUploadWS] receive error: \(error.localizedDescription) | \(error)")
                }
                break
            }
        }
    }

    private func makeWebSocketTask(url: URL) -> URLSessionWebSocketTask {
        var request = URLRequest(url: url)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = TokenStorage.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return session.webSocketTask(with: request)
    }

    private func websocketURL(tripId: String, sessionId: String) -> URL {
        let base = hostURL()
        var components = URLComponents()
        components.scheme = base.scheme
        components.host = base.host
        components.port = base.port
        components.path = "/api/v1/trips/\(tripId)/pin-uploads/\(sessionId)/ws"
        return components.url ?? base
    }

    private func hostURL() -> URL {
        if let url = URL(string: PinzLaunchArgs.websocketURLString) {
            return url
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

extension PinUploadWebSocketClient {

    static func parseEvent(from data: Data) -> PinUploadWSEvent {
        guard let envelope = try? JSONDecoder().decode(WSEnvelope.self, from: data) else {
            print("[PinUploadWS] parse: envelope decode failed")
            return .unknown
        }

        let eventType = normalize(envelope.resolvedType)

        switch eventType {
        case normalize("PIN_UPLOAD_PROCESSING_COMPLETED"):
            if let p = try? JSONDecoder().decode(WSWithPayload<ProcessingCompletedPayload>.self, from: data) {
                return .processingCompleted(
                    sessionId: p.payload.sessionId,
                    targetPinId: p.payload.targetPinId,
                    processingStatus: p.payload.processingStatus
                )
            }
            print("[PinUploadWS] parse: PIN_UPLOAD_PROCESSING_COMPLETED payload decode failed")

        default:
            print("[PinUploadWS] parse: unknown event type=\(envelope.resolvedType)")
        }

        return .unknown
    }

    static func describe(_ event: PinUploadWSEvent) -> String {
        switch event {
        case let .processingCompleted(sessionId, targetPinId, status):
            return "processingCompleted(sessionId=\(sessionId), targetPinId=\(targetPinId ?? "nil"), status=\(status))"
        case .unknown:
            return "unknown"
        }
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

private struct ProcessingCompletedPayload: Decodable {
    let tripId: String?
    let sessionId: String
    let targetPinId: String?
    let processingStatus: String

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case sessionId = "session_id"
        case targetPinId = "target_pin_id"
        case processingStatus = "processing_status"
    }
}
