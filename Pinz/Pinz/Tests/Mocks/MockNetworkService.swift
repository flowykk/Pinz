import Foundation
import PinzNetworking
import PinzDomain

final class MockNetworkService: NetworkServiceProtocol {
    var submitEmailResult: Result<SubmitEmailResponse, Error> = .success(
        SubmitEmailResponse(isRegistered: false, registrationId: "reg-123")
    )
    var verifyEmailResult: Result<SuccessResponse, Error> = .success(SuccessResponse(success: true))
    var passkeyLoginBeginResult: Result<PasskeyOptionsResponse, Error> = .success(
        PasskeyOptionsResponse(optionsJson: "")
    )
    var passkeyLoginFinishResult: Result<UserTokensResponse, Error> = .success(
        UserTokensResponse(accessToken: "access", refreshToken: "refresh")
    )
    var passkeyRegisterBeginResult: Result<PasskeyOptionsResponse, Error> = .success(
        PasskeyOptionsResponse(optionsJson: "")
    )
    var passkeyRegisterFinishResult: Result<UserTokensResponse, Error> = .success(
        UserTokensResponse(accessToken: "access", refreshToken: "refresh")
    )
    var refreshTokenResult: Result<RefreshTokenResponse, Error> = .success(
        RefreshTokenResponse(accessToken: "access")
    )
    var logoutResult: Result<SuccessResponse, Error> = .success(SuccessResponse(success: true))

    func submitEmail(email: String) async throws -> SubmitEmailResponse {
        try submitEmailResult.get()
    }

    func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessResponse {
        try verifyEmailResult.get()
    }

    func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsResponse {
        try passkeyLoginBeginResult.get()
    }

    func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensResponse {
        try passkeyLoginFinishResult.get()
    }

    func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsResponse {
        try passkeyRegisterBeginResult.get()
    }

    func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensResponse {
        try passkeyRegisterFinishResult.get()
    }

    func refreshToken(refreshToken: String) async throws -> RefreshTokenResponse {
        try refreshTokenResult.get()
    }

    func logout(refreshToken: String) async throws -> SuccessResponse {
        try logoutResult.get()
    }
}
