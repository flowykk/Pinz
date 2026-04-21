import Foundation
import PinzBase

public enum TripReviewWaitError: Error {
    case timeout
    case cancelled
    case webSocket(message: String)
}

final class TripCreationWebSocketClient {
    private enum LogLevel: String {
        case info = "INFO"
        case warn = "WARN"
        case error = "ERROR"
    }

    private struct TripReviewSocketPayload: Decodable {
        let tripId: String
        let status: String

        private enum CodingKeys: String, CodingKey {
            case tripId
            case tripIdSnake = "trip_id"
            case status
        }

        init(from decoder: Decoder) throws {
            let container = try decoder.container(keyedBy: CodingKeys.self)
            let snakeCaseTripId = try container.decodeIfPresent(String.self, forKey: .tripIdSnake)
            let camelCaseTripId = try container.decodeIfPresent(String.self, forKey: .tripId)
            tripId = camelCaseTripId?.isEmpty == false ? camelCaseTripId ?? "" : snakeCaseTripId ?? ""
            status = try container.decode(String.self, forKey: .status)
        }
    }

    private struct TripReviewSocketEvent: Decodable {
        let event: String
        let payload: TripReviewSocketPayload
    }

    private enum TargetEvent {
        static let processingCompleted = normalize("TRIP_PROCESSING_COMPLETED")
        static let draftFinalReview = normalize("DRAFT_FINAL_REVIEW")
    }

    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    func waitForTripProcessingCompleted(tripId: String, timeout: TimeInterval) async throws {
        let start = Date()
        let hasAuth = TokenStorage.shared.accessToken != nil
        let url = websocketURL(for: tripId)

        Self.log(
            "Connecting",
            level: .info,
            details: [
                "URL: \(url.absoluteString)",
                "tripId: \(tripId)",
                "timeout: \(String(format: "%.2f", timeout))s",
                "hasAuth: \(hasAuth)"
            ]
        )

        var request = URLRequest(url: url)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = TokenStorage.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let webSocketTask = session.webSocketTask(with: request)
        webSocketTask.resume()
        Self.log("Connected", level: .info, details: ["tripId: \(tripId)"])
        Self.log("Subscribed", level: .info, details: ["tripId: \(tripId)"])

        var stopReason = "completed"
        var stopLevel: LogLevel = .info

        defer {
            Self.log(
                "Stopping",
                level: stopLevel,
                details: [
                    "reason: \(stopReason)",
                    "elapsed: \(Self.formattedElapsed(since: start))"
                ]
            )
            webSocketTask.cancel(
                with: URLSessionWebSocketTask.CloseCode.normalClosure,
                reason: nil as Data?
            )
        }

        do {
            try await withThrowingTaskGroup(of: Void.self) { group in
                group.addTask {
                    try await Self.waitForCompletionEvent(
                        webSocketTask: webSocketTask,
                        expectedTripId: tripId
                    )
                }

                group.addTask {
                    if timeout <= 0 { throw TripReviewWaitError.timeout }
                    do {
                        try await Task.sleep(nanoseconds: UInt64(timeout * 1_000_000_000))
                    } catch {
                        throw TripReviewWaitError.cancelled
                    }
                    Self.log(
                        "Wait timeout exceeded",
                        level: .warn,
                        details: ["tripId: \(tripId)", "timeout: \(Self.formattedElapsed(since: start))"]
                    )
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
        } catch {
            stopLevel = .warn
            stopReason = Self.debugMessage(for: error)
            throw error
        }
    }

    private static func waitForCompletionEvent(
        webSocketTask: URLSessionWebSocketTask,
        expectedTripId: String
    ) async throws {
        while !Task.isCancelled {
            do {
                let message = try await webSocketTask.receive()
                let data: Data

                switch message {
                case .string(let string):
                    data = Data(string.utf8)
                    Self.log(
                        "Received raw message",
                        details: [
                            "tripId: \(expectedTripId)",
                            "type: .string",
                            "length: \(data.count)"
                        ]
                    )
                case .data(let incoming):
                    data = incoming
                    Self.log(
                        "Received raw message",
                        details: [
                            "tripId: \(expectedTripId)",
                            "type: .data",
                            "length: \(data.count)"
                        ]
                    )
                @unknown default:
                    Self.log(
                        "Received raw message",
                        level: .warn,
                        details: [
                            "tripId: \(expectedTripId)",
                            "type: unknown"
                        ]
                    )
                    continue
                }

                do {
                    let decoded = try JSONDecoder().decode(TripReviewSocketEvent.self, from: data)
                    let event = Self.normalize(decoded.event)
                    let status = Self.normalize(decoded.payload.status)
                    let payloadTripId = decoded.payload.tripId

                    log(
                        "Parsed event",
                        level: .info,
                        details: [
                            "event: \(decoded.event)",
                            "trip_id: \(payloadTripId)",
                            "status: \(decoded.payload.status)"
                        ]
                    )

                    guard payloadTripId == expectedTripId else {
                        log(
                            "Ignored event for different trip/event",
                            level: .warn,
                            details: [
                                "tripId: \(expectedTripId)",
                                "receivedTripId: \(payloadTripId)",
                                "event: \(decoded.event)"
                            ]
                        )
                        continue
                    }

                    guard event == TargetEvent.processingCompleted else {
                        log(
                            "Ignored event for different trip/event",
                            level: .warn,
                            details: [
                                "tripId: \(expectedTripId)",
                                "event: \(decoded.event)",
                                "status: \(decoded.payload.status)"
                            ]
                        )
                        continue
                    }

                    guard status == TargetEvent.draftFinalReview else {
                        log(
                            "Ignored event for different trip/event",
                            level: .warn,
                            details: [
                                "tripId: \(expectedTripId)",
                                "status: \(decoded.payload.status)"
                            ]
                        )
                        continue
                    }

                    return
                } catch {
                    Self.log(
                        "decode error",
                        level: .warn,
                        details: [
                            "tripId: \(expectedTripId)",
                            "error: \(error.localizedDescription)"
                        ]
                    )
                }
            } catch {
                if error is CancellationError || Task.isCancelled {
                    Self.log(
                        "Task cancellation",
                        level: .warn,
                        details: ["tripId: \(expectedTripId)"]
                    )
                    throw TripReviewWaitError.cancelled
                }

                Self.log(
                    "network/protocol error",
                    level: .error,
                    details: [
                        "tripId: \(expectedTripId)",
                        "error: \(error.localizedDescription)"
                    ]
                )
                throw TripReviewWaitError.webSocket(message: error.localizedDescription)
            }
        }

        throw TripReviewWaitError.cancelled
    }

    private func websocketURL(for tripId: String) -> URL {
        let base = commandLineHostURL()
        var components = URLComponents()
        components.scheme = base.scheme
        components.host = base.host
        components.port = base.port
        components.path = "/api/v1/trips/creation/\(tripId)/review/ws"
        return components.url ?? base
    }

    private func commandLineHostURL() -> URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "ws://localhost:8080")!
        }
        return URL(string: "wss://pinz.website")!
    }

    private static func normalize(_ raw: String) -> String {
        raw
            .uppercased()
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: "_", with: "")
    }

    private static func formattedElapsed(since start: Date) -> String {
        String(format: "%.2fs", Date().timeIntervalSince(start))
    }

    private static func debugMessage(for error: Error) -> String {
        if let tripError = error as? TripReviewWaitError {
            switch tripError {
            case .timeout:
                return "timeout"
            case .cancelled:
                return "cancelled"
            case let .webSocket(message):
                return "webSocket: \(message)"
            }
        }
        return "\(error)"
    }

    private static func log(_ message: String, level: LogLevel = .info, details: [String] = []) {
        #if DEBUG
        let title = "┌─ [WS][\(level.rawValue)] \(message)"
        let body = details.map { "│  \($0)" }.joined(separator: "\n")
        if body.isEmpty {
            print("\(title)\n└─────────────────────────")
        } else {
            print("\(title)\n\(body)\n└─────────────────────────")
        }
        #endif
    }
}
