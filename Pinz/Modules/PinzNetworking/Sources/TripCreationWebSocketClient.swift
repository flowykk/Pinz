import Foundation
import PinzBase

public enum TripReviewWaitError: Error {
    case timeout
    case cancelled
    case webSocket(message: String)
}

final class TripCreationWebSocketClient {
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
        let url = websocketURL(for: tripId)

        var request = URLRequest(url: url)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = TokenStorage.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let webSocketTask = session.webSocketTask(with: request)
        webSocketTask.resume()

        defer {
            webSocketTask.cancel(
                with: URLSessionWebSocketTask.CloseCode.normalClosure,
                reason: nil as Data?
            )
        }

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
                case .data(let incoming):
                    data = incoming
                @unknown default:
                    continue
                }

                do {
                    let decoded = try JSONDecoder().decode(TripReviewSocketEvent.self, from: data)
                    let event = Self.normalize(decoded.event)
                    let status = Self.normalize(decoded.payload.status)
                    let payloadTripId = decoded.payload.tripId

                    guard payloadTripId == expectedTripId else {
                        continue
                    }

                    guard event == TargetEvent.processingCompleted else {
                        continue
                    }

                    guard status == TargetEvent.draftFinalReview else {
                        continue
                    }

                    return
                } catch {
                    // no-op
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
        if let url = URL(string: PinzLaunchArgs.websocketURLString) {
            return url
        }
        return URL(string: "wss://pinz.website")!
    }

    private static func normalize(_ raw: String) -> String {
        raw
            .uppercased()
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: "_", with: "")
    }

}
