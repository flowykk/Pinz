// swiftlint:disable function_parameter_count file_length type_body_length
import SwiftUI
import Moya
import Foundation
import PinzBase
import PinzDomain

public protocol NetworkServiceProtocol {
    func checkEmail(email: String) async throws -> SuccessResponse

    func register(email: String) async throws -> RegisterResponse
    func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessResponse
    func finishRegister(password: String, registrationId: String, username: String) async throws -> UserTokensResponse

    func login(email: String, password: String) async throws -> UserTokensResponse

}

public final class NetworkService: NetworkServiceProtocol {
    private let provider = NetworkProvider<PinzAPI>(stub: true, stubDelay: 1)

    public init() {}

    public func checkEmail(email: String) async throws -> SuccessResponse {
        try await provider.request(
            .checkEmail(
                email: email
            ),
            type: SuccessResponse.self
        )
    }

    public func register(email: String) async throws -> RegisterResponse {
        try await provider.request(
            .register(
                email: email
            ),
            type: RegisterResponse.self
        )
    }

    public func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessResponse {
        try await provider.request(
            .verifyEmail(
                registrationId: registrationId,
                verificationCode: verificationCode
            ),
            type: SuccessResponse.self
        )
    }

    public func finishRegister(
        password: String,
        registrationId: String,
        username: String
    ) async throws -> UserTokensResponse {
        try await provider.request(
            .finishRegister(
                password: password,
                registrationId: registrationId,
                username: username
            ),
            type: UserTokensResponse.self
        )
    }

    public func login(email: String, password: String) async throws -> UserTokensResponse {
        try await provider.request(
            .login(
                email: email,
                password: password
            ),
            type: UserTokensResponse.self
        )
    }
}
// swiftlint:enable function_parameter_count file_length type_body_length
