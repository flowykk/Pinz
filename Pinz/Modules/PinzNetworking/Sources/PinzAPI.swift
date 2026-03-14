import SwiftUI
import Moya
import Foundation
import PinzBase

enum PinzAPI {
    case submitEmail(email: String)
    case verifyEmail(registrationId: String, verificationCode: String)

    case passkeyLoginBegin(email: String)
    case passkeyLoginFinish(email: String, credentialJSON: String)

    case passkeyRegisterBegin(registrationId: String, username: String)
    case passkeyRegisterFinish(registrationId: String, credentialJSON: String)

    case refreshToken(refreshToken: String)
    case logout(refreshToken: String)
}

extension PinzAPI: TargetType {
    var baseURL: URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "http://localhost:8080")!
        }
        return URL(string: "https://pinz.website/api/v1")!
    }

    var path: String {
        switch self {
        case .submitEmail:
            return "/auth/email"
        case .verifyEmail:
            return "/auth/verify-email"
        case .passkeyLoginBegin:
            return "/auth/passkey/login/begin"
        case .passkeyLoginFinish:
            return "/auth/passkey/login/finish"
        case .passkeyRegisterBegin:
            return "/auth/passkey/register/begin"
        case .passkeyRegisterFinish:
            return "/auth/passkey/register/finish"
        case .refreshToken:
            return "/auth/refresh"
        case .logout:
            return "/auth/logout"
        }
    }

    var method: Moya.Method {
        .post
    }

    var task: Moya.Task {
        switch self {
        case let .submitEmail(email):
            return jsonRequest(["email": email])
        case let .verifyEmail(registrationId, verificationCode):
            return jsonRequest(["registration_id": registrationId, "verification_code": verificationCode])
        case let .passkeyLoginBegin(email):
            return jsonRequest(["email": email])
        case let .passkeyLoginFinish(email, credentialJSON):
            return jsonRequest(["email": email, "credential_json": credentialJSON])
        case let .passkeyRegisterBegin(registrationId, username):
            return jsonRequest(["registration_id": registrationId, "username": username])
        case let .passkeyRegisterFinish(registrationId, credentialJSON):
            return jsonRequest(["registration_id": registrationId, "credential_json": credentialJSON])
        case let .refreshToken(refreshToken):
            return jsonRequest(["refresh_token": refreshToken])
        case let .logout(refreshToken):
            return jsonRequest(["refresh_token": refreshToken])
        }
    }

    var headers: [String: String]? {
        return ["Content-Type": "application/json"]
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
        case .submitEmail:
            result = """
            {"is_registered": false, "registration_id": "550e8400-e29b-41d4-a716-446655440000"}
            """
        case .verifyEmail:
            result = """
            {"success": true}
            """
        case .passkeyLoginBegin, .passkeyRegisterBegin:
            result = """
            {"options_json": "eyJjaGFsbGVuZ2UiOiJ0ZXN0In0="}
            """
        case .passkeyLoginFinish, .passkeyRegisterFinish:
            result = """
            {"access_token": "stub_access_token", "refresh_token": "stub_refresh_token"}
            """
        case .refreshToken:
            result = """
            {"access_token": "stub_new_access_token"}
            """
        case .logout:
            result = """
            {"success": true}
            """
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
