import Moya
import Foundation

final class NetworkProvider<T: TargetType> {
    private let provider: MoyaProvider<T>

    init(
        stub: Bool = false,
        stubDelay: TimeInterval = 0,
        timeout: TimeInterval = 5
    ) {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = timeout

        self.provider = MoyaProvider<T>(
            stubClosure: stub ? MoyaProvider.delayedStub(stubDelay) : MoyaProvider.neverStub,
            session: Session(configuration: configuration)
        )
    }

    func requestRaw(_ target: T) async throws -> Moya.Response {
        Self.logRequest(target: target)
        return try await withCheckedThrowingContinuation { continuation in
            provider.request(target) { result in
                switch result {
                case .success(let response):
                    Self.logResponse(target: target, response: response)
                    guard response.statusCode >= 200 && response.statusCode < 300 else {
                        if response.statusCode == 401, let pinzTarget = target as? PinzAPI {
                            PinzSessionInvalidation.handleUnauthorizedFromAPI(for: pinzTarget)
                        }
                        continuation.resume(throwing: HTTPError(statusCode: response.statusCode, reason: response.debugDescription))
                        return
                    }
                    continuation.resume(returning: response)
                case .failure(let error):
                    Self.logError(target: target, error: error)
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    func request<D: Decodable>(_ target: T, type: D.Type) async throws -> D {
        Self.logRequest(target: target)
        return try await withCheckedThrowingContinuation { continuation in
            provider.request(target) { result in
                switch result {
                case .success(let response):
                    Self.logResponse(target: target, response: response)
                    guard response.statusCode >= 200 && response.statusCode < 300 else {
                        if response.statusCode == 401, let pinzTarget = target as? PinzAPI {
                            PinzSessionInvalidation.handleUnauthorizedFromAPI(for: pinzTarget)
                        }
                        let error = HTTPError(statusCode: response.statusCode, reason: response.debugDescription)
                        continuation.resume(throwing: error)
                        return
                    }
                    do {
                        let decoded = try JSONDecoder().decode(D.self, from: response.data)
                        continuation.resume(returning: decoded)
                    } catch {
                        continuation.resume(throwing: error)
                    }
                case .failure(let error):
                    Self.logError(target: target, error: error)
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    private static func logRequest(target: T) {
        var bodyString = "-"
        if case .requestParameters(let parameters, _) = target.task {
            if let string = Self.stringified(parameters) {
                bodyString = string
            } else {
                bodyString = "Unserializable request parameters: \(String(describing: parameters))"
            }
        }
        print("""
        ┌─ → \(target.method.rawValue) \(target.baseURL)\(target.path)
        │  Request:
        \(bodyString.split(separator: "\n").map { "│  \($0)" }.joined(separator: "\n"))
        │
        """)
    }

    private static func stringified(_ parameters: [String: Any]) -> String? {
        if JSONSerialization.isValidJSONObject(parameters),
           let data = try? JSONSerialization.data(withJSONObject: parameters, options: .prettyPrinted),
           let string = String(data: data, encoding: .utf8) {
            return string
        }
        return nil
    }

    private static func logResponse(target: T, response: Moya.Response) {
        let body = (try? JSONSerialization.jsonObject(with: response.data))
            .flatMap { try? JSONSerialization.data(withJSONObject: $0, options: .prettyPrinted) }
            .flatMap { String(data: $0, encoding: .utf8) } ?? String(data: response.data, encoding: .utf8) ?? "-"
        print("""
        ┌─ \(target.method.rawValue) \(target.baseURL)\(target.path)
        │  Status: \(response.statusCode)
        │  Response:
        \(body.split(separator: "\n").map { "│  \($0)" }.joined(separator: "\n"))
        └─────────────────────────
        """)
    }

    private static func logError(target: T, error: Error) {
        print("""
        ┌─ \(target.method.rawValue) \(target.baseURL)\(target.path)
        │  Error: \(error.localizedDescription)
        └─────────────────────────
        """)
    }
}
