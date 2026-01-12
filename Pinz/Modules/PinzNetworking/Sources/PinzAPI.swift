import SwiftUI
import Moya
import Foundation
import PinzBase

enum PinzAPI {
    case checkEmail(email: String)

    case register(email: String)
    case verifyEmail(registrationId: String, verificationCode: String)
    case finishRegister(password: String, registrationId: String, username: String)

    case login(email: String, password: String)
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
        case .checkEmail:
            return "/auth/check-email"
        case .register:
            return "/auth/register"
        case .verifyEmail:
            return "/auth/verify-email"
        case .finishRegister:
            return "/auth/finish-register"
        case .login:
            return "/auth/login"
        }
    }

    var method: Moya.Method {
        switch self {
        case .checkEmail,
                .register,
                .verifyEmail,
                .finishRegister,
                .login:
            return .post
        }
    }

    var task: Moya.Task {
        switch self {
        case let .checkEmail(email):
            return jsonRequest(["email": email])
        case let .register(email):
            return jsonRequest(["email": email])
        case let .verifyEmail(registrationId, verificationCode):
            return jsonRequest(["registration_id": registrationId, "verification_code": verificationCode])
        case let .finishRegister(password, registrationId, username):
            return jsonRequest([
                "password": password,
                "registration_id": registrationId,
                "username": username
            ])
        case let .login(email, password):
            return jsonRequest(["email": email, "password": password])
        }
    }

    var headers: [String: String]? {
        switch self {
        default:
            return ["Content-Type": "application/json"]
        }

        /*
         return [
             "Authorization": "Bearer x",
             "Content-Type": "application/json"
         ]
         */
    }

    private func jsonRequest(_ parameters: [String: Any]) -> Moya.Task {
        .requestParameters(parameters: parameters, encoding: JSONEncoding.default)
    }
}

// MARK: - Mocks
extension PinzAPI {
    var sampleData: Data {
        let result: String
        switch self {
        case .checkEmail:
            result = """
            {"success": true}
            """
        case .register:
            result = """
            {"name": "Test"}
            """
        default:
            result = ""
        }

        return result.data(using: .utf8) ?? Data()
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
