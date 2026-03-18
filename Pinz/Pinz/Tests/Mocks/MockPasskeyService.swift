import Foundation
import PinzBase

@MainActor
final class MockPasskeyService: PasskeyServiceProtocol {
    var assertionResult: Result<String, Error> = .success("mock_credential_json")
    var attestationResult: Result<String, Error> = .success("mock_credential_json")

    func performAssertion(optionsJSON: String) async throws -> String {
        try assertionResult.get()
    }

    func performAttestation(optionsJSON: String) async throws -> String {
        try attestationResult.get()
    }
}
