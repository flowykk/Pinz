import Foundation
import Vapor

struct ProfileController: RouteCollection {
    let responseFactory: ProfileResponseFactory

    func boot(routes: RoutesBuilder) throws {
        routes.get("health", use: health)

        let api = routes.grouped("api", "v1", "profile")
        api.get(use: getProfile)
        api.patch(use: patchProfile)
        api.post("change-email", use: requestEmailChange)
        api.post("confirm-email", use: confirmEmailChange)
    }

    private func health(_ req: Request) -> Response {
        let response = Response(status: .ok)
        response.body = .init(string: "ok")
        return response
    }

    private func getProfile(_: Request) async -> Response {
        await responseFactory.profileResponse()
    }

    private func patchProfile(req: Request) async throws -> Response {
        let body = try req.content.decode(MockUpdateProfileRequest.self)
        return await responseFactory.updateProfileResponse(for: body)
    }

    private func requestEmailChange(req: Request) async throws -> Response {
        let body = try req.content.decode(MockChangeEmailRequest.self)
        return await responseFactory.requestEmailChangeResponse(for: body)
    }

    private func confirmEmailChange(req: Request) async throws -> Response {
        let body = try req.content.decode(MockConfirmEmailRequest.self)
        guard let response = await responseFactory.confirmEmailChangeResponse(for: body) else {
            throw Abort(.badRequest, reason: "Invalid verification code")
        }
        return response
    }
}
