import SwiftUI
import Moya
import Foundation
import Base

enum PinzAPI {
    case register(email: String)
}

extension PinzAPI: TargetType {
    var baseURL: URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "http://localhost:8080")!
        }
        return URL(string: "https://pinzapp.ru/api")!
    }

    var path: String {
        switch self {
        case .register:
            return "/auth/register"
        }
    }

    var method: Moya.Method {
        switch self {
        case .register:
            return .post
        }
    }

    var task: Moya.Task {
        switch self {
        case let .register(email):
            return jsonRequest(["email": email])
        }
    }

    var headers: [String: String]? {
        switch self {
        case .register:
            return [
                "Authorization": "Bearer x",
                "Content-Type": "application/json"
            ]
        default:
            return ["Content-Type": "application/json"]
        }
    }

    private func jsonRequest(_ parameters: [String: Any]) -> Moya.Task {
        .requestParameters(parameters: parameters, encoding: JSONEncoding.default)
    }
}

extension PinzAPI {
    var sampleData: Data {
        switch self {
        case .register:
            return """
            {"name": "Test"}
            """.data(using: .utf8)!
        default:
            return Data()
        }
    }
}

extension String {
    func defaultUTF8Data() -> Data? {
        self.data(using: .utf8)
    }

    func toISO8601String() -> String? {
        let inputFormatter = DateFormatter()
        inputFormatter.locale = Locale(identifier: "en_US_POSIX")
        inputFormatter.dateFormat = "dd.MM.yyyy HH:mm"

        guard let date = inputFormatter.date(from: self) else { return nil }

        let isoFormatter = ISO8601DateFormatter()
        isoFormatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return isoFormatter.string(from: date)
    }
}
