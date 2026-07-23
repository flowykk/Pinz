import AuthenticationServices
import Foundation
import UIKit

// MARK: - PasskeyServiceProtocol

@MainActor
public protocol PasskeyServiceProtocol {
    func performAssertion(optionsJSON: String) async throws -> String
    func performAttestation(optionsJSON: String) async throws -> String
}

// MARK: - PasskeyService

@MainActor
public final class PasskeyService: NSObject, PasskeyServiceProtocol {

    private var continuation: CheckedContinuation<ASAuthorization, Error>?

    public override init() {
        super.init()
    }

    public func performAssertion(optionsJSON: String) async throws -> String {
        let options = try parseOptions(optionsJSON)

        guard let challenge = options["challenge"] as? String,
              let challengeData = Data(base64URLEncoded: challenge) else {
            throw PasskeyError.invalidOptions
        }

        let rpId = options["rpId"] as? String ?? ""

        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(relyingPartyIdentifier: rpId)
        let request = provider.createCredentialAssertionRequest(challenge: challengeData)

        if let allowCredentials = options["allowCredentials"] as? [[String: Any]] {
            request.allowedCredentials = allowCredentials.compactMap { cred in
                guard let id = cred["id"] as? String,
                      let idData = Data(base64URLEncoded: id) else { return nil }
                return ASAuthorizationPlatformPublicKeyCredentialDescriptor(credentialID: idData)
            }
        }

        let authorization = try await performRequest(request)

        guard let assertion = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialAssertion else {
            throw PasskeyError.unexpectedCredentialType
        }

        return buildAssertionJSON(assertion)
    }

    public func performAttestation(optionsJSON: String) async throws -> String {
        let options = try parseOptions(optionsJSON)

        guard let challenge = options["challenge"] as? String,
              let challengeData = Data(base64URLEncoded: challenge) else {
            throw PasskeyError.invalidOptions
        }

        let rpId: String
        if let rp = options["rp"] as? [String: Any] {
            rpId = rp["id"] as? String ?? ""
        } else {
            rpId = options["rpId"] as? String ?? ""
        }

        guard let user = options["user"] as? [String: Any],
              let userId = user["id"] as? String,
              let userIdData = Data(base64URLEncoded: userId),
              let userName = user["name"] as? String else {
            throw PasskeyError.invalidOptions
        }

        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(relyingPartyIdentifier: rpId)
        let request = provider.createCredentialRegistrationRequest(
            challenge: challengeData,
            name: userName,
            userID: userIdData
        )

        let authorization = try await performRequest(request)

        guard let registration = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialRegistration else {
            throw PasskeyError.unexpectedCredentialType
        }

        return buildRegistrationJSON(registration)
    }
}

// MARK: - Private

private extension PasskeyService {

    func performRequest(_ request: ASAuthorizationRequest) async throws -> ASAuthorization {
        let controller = ASAuthorizationController(authorizationRequests: [request])
        controller.delegate = self
        controller.presentationContextProvider = self

        return try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
            controller.performRequests()
        }
    }

    func parseOptions(_ base64String: String) throws -> [String: Any] {
        guard let data = Data(base64Encoded: base64String),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw PasskeyError.invalidOptions
        }
        // Server may wrap options in a "publicKey" key (WebAuthn JS API format)
        if let publicKey = json["publicKey"] as? [String: Any] {
            return publicKey
        }
        return json
    }

    func buildAssertionJSON(_ assertion: ASAuthorizationPlatformPublicKeyCredentialAssertion) -> String {
        let credential: [String: Any] = [
            "id": assertion.credentialID.base64URLEncodedString(),
            "rawId": assertion.credentialID.base64URLEncodedString(),
            "type": "public-key",
            "response": [
                "authenticatorData": assertion.rawAuthenticatorData.base64URLEncodedString(),
                "clientDataJSON": assertion.rawClientDataJSON.base64URLEncodedString(),
                "signature": assertion.signature.base64URLEncodedString(),
                "userHandle": assertion.userID.base64URLEncodedString()
            ],
            "clientExtensionResults": [String: Any]()
        ]

        guard let jsonData = try? JSONSerialization.data(withJSONObject: credential) else { return "" }
        return jsonData.base64EncodedString()
    }

    func buildRegistrationJSON(_ registration: ASAuthorizationPlatformPublicKeyCredentialRegistration) -> String {
        var response: [String: Any] = [
            "clientDataJSON": registration.rawClientDataJSON.base64URLEncodedString()
        ]
        if let attestation = registration.rawAttestationObject {
            response["attestationObject"] = attestation.base64URLEncodedString()
        }

        let credential: [String: Any] = [
            "id": registration.credentialID.base64URLEncodedString(),
            "rawId": registration.credentialID.base64URLEncodedString(),
            "type": "public-key",
            "response": response,
            "clientExtensionResults": [String: Any]()
        ]

        guard let jsonData = try? JSONSerialization.data(withJSONObject: credential) else { return "" }
        return jsonData.base64EncodedString()
    }
}

// MARK: - ASAuthorizationControllerDelegate

extension PasskeyService: ASAuthorizationControllerDelegate {
    nonisolated public func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithAuthorization authorization: ASAuthorization
    ) {
        Task { @MainActor in
            continuation?.resume(returning: authorization)
            continuation = nil
        }
    }

    nonisolated public func authorizationController(
        controller: ASAuthorizationController,
        didCompleteWithError error: Error
    ) {
        let authError = error as? ASAuthorizationError
        print("[PasskeyService] Error code=\(authError?.code.rawValue ?? -1): \(error.localizedDescription)")
        Task { @MainActor in
            continuation?.resume(throwing: error)
            continuation = nil
        }
    }
}

// MARK: - ASAuthorizationControllerPresentationContextProviding

extension PasskeyService: ASAuthorizationControllerPresentationContextProviding {
    nonisolated public func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        MainActor.assumeIsolated {
            guard let scene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
                  let window = scene.windows.first(where: { $0.isKeyWindow }) else {
                return ASPresentationAnchor()
            }
            return window
        }
    }
}

// MARK: - PasskeyError

public enum PasskeyError: LocalizedError {
    case invalidOptions
    case unexpectedCredentialType

    public var errorDescription: String? {
        switch self {
        case .invalidOptions:
            return "Invalid passkey options from server"
        case .unexpectedCredentialType:
            return "Unexpected credential type"
        }
    }
}
