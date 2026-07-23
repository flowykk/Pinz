import Foundation
import Vapor

struct PinUploadController: RouteCollection {
    let responseFactory: PinUploadResponseFactory

    func boot(routes: RoutesBuilder) throws {
        let api = routes.grouped("api", "v1", "trips")
        api.post(":tripId", "pin-uploads", "start", use: start)
        api.post(":tripId", "pin-uploads", ":sessionId", "commit-upload", use: commit)
        api.post(":tripId", "pin-uploads", ":sessionId", "process", use: process)
        api.get(":tripId", "pin-uploads", ":sessionId", "review", use: review)
        api.post(":tripId", "pin-uploads", ":sessionId", "finalize", use: finalize)
        api.post(":tripId", "pin-uploads", ":sessionId", "cancel", use: cancel)

        routes.on(.PUT, "mock-upload", ":clientId", body: .collect, use: upload)
    }

    private func start(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let body = try req.content.decode(MockPinUploadStartRequest.self)
        return await responseFactory.start(tripId: tripId, request: body)
    }

    private func upload(_: Request) async -> Response {
        await responseFactory.upload()
    }

    private func commit(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let sessionId = req.parameters.get("sessionId") ?? ""
        let body = try req.content.decode(MockPinUploadCommitRequest.self)
        return await responseFactory.commit(tripId: tripId, sessionId: sessionId, request: body)
    }

    private func process(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let sessionId = req.parameters.get("sessionId") ?? ""
        return await responseFactory.process(tripId: tripId, sessionId: sessionId)
    }

    private func review(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let sessionId = req.parameters.get("sessionId") ?? ""
        return await responseFactory.review(tripId: tripId, sessionId: sessionId)
    }

    private func finalize(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let sessionId = req.parameters.get("sessionId") ?? ""
        let body = try req.content.decode(MockPinUploadFinalizeRequest.self)
        return await responseFactory.finalize(tripId: tripId, sessionId: sessionId, request: body)
    }

    private func cancel(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let sessionId = req.parameters.get("sessionId") ?? ""
        return await responseFactory.cancel(tripId: tripId, sessionId: sessionId)
    }
}
