// swiftlint:disable function_parameter_count file_length type_body_length
import SwiftUI
import Moya
import Foundation
import Base

public struct RegisterResponse: Codable {
    let name: String
}

public protocol NetworkServiceProtocol {
    func register(email: String) async throws -> RegisterResponse
}

public final class NetworkService: NetworkServiceProtocol {
    private let provider = NetworkProvider<PinzAPI>(stub: true, stubDelay: 1)

    public init() {}

    public func register(email: String) async throws -> RegisterResponse {
        try await provider.request(
            .register(
                email: email
            ),
            type: RegisterResponse.self
        )
    }
}
// swiftlint:enable function_parameter_count file_length type_body_length
