// swiftlint:disable function_parameter_count file_length type_body_length
import SwiftUI
import Moya
import Foundation
import PinzBase
import PinzDomain

public protocol NetworkServiceProtocol {
    func submitEmail(email: String) async throws -> SubmitEmailResponse
    func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessResponse

    func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsResponse
    func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensResponse

    func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsResponse
    func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensResponse

    func refreshToken(refreshToken: String) async throws -> RefreshTokenResponse
    func logout(refreshToken: String) async throws -> SuccessResponse
}

public final class NetworkService: NetworkServiceProtocol {
    private let provider = NetworkProvider<PinzAPI>()

    public init() {}

    public func submitEmail(email: String) async throws -> SubmitEmailResponse {
        try await provider.request(
            .submitEmail(email: email),
            type: SubmitEmailResponse.self
        )
    }

    public func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessResponse {
        try await provider.request(
            .verifyEmail(registrationId: registrationId, verificationCode: verificationCode),
            type: SuccessResponse.self
        )
    }

    public func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsResponse {
        try await provider.request(
            .passkeyLoginBegin(email: email),
            type: PasskeyOptionsResponse.self
        )
    }

    public func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensResponse {
        try await provider.request(
            .passkeyLoginFinish(email: email, credentialJSON: credentialJSON),
            type: UserTokensResponse.self
        )
    }

    public func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsResponse {
        try await provider.request(
            .passkeyRegisterBegin(registrationId: registrationId, username: username),
            type: PasskeyOptionsResponse.self
        )
    }

    public func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensResponse {
        try await provider.request(
            .passkeyRegisterFinish(registrationId: registrationId, credentialJSON: credentialJSON),
            type: UserTokensResponse.self
        )
    }

    public func refreshToken(refreshToken: String) async throws -> RefreshTokenResponse {
        try await provider.request(
            .refreshToken(refreshToken: refreshToken),
            type: RefreshTokenResponse.self
        )
    }

    public func logout(refreshToken: String) async throws -> SuccessResponse {
        try await provider.request(
            .logout(refreshToken: refreshToken),
            type: SuccessResponse.self
        )
    }
}
// swiftlint:enable function_parameter_count file_length type_body_length
