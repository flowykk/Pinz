import Foundation
import Vapor

struct TripCreationController: RouteCollection {
    let responseFactory: TripCreationResponseFactory

    func boot(routes: RoutesBuilder) throws {
        routes.get("health", use: health)

        let api = routes.grouped("api", "v1", "trips", "creation")
        api.post("start", use: create)
        api.post(":tripId", "media", "process-grouping", use: processGrouping)
        api.post(":tripId", "apply-groups-and-process", use: applyGroupsAndProcess)
        api.get(":tripId", "review", use: review)
        api.post(":tripId", "finalize", use: finalize)

        routes.on(.PUT, "mock-trip-creation-upload", ":clientId", body: .collect, use: upload)
    }

    private func health(_: Request) -> Response {
        let response = Response(status: .ok)
        response.body = .init(string: "ok")
        return response
    }

    private func create(_ req: Request) async throws -> Response {
        let body = try req.content.decode(MockCreateTripRequest.self)
        return await responseFactory.create(request: body)
    }

    private func upload(_: Request) async -> Response {
        await responseFactory.upload()
    }

    private func processGrouping(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let body = try req.content.decode(MockProcessMediaGroupingRequest.self)
        return await responseFactory.processGrouping(tripId: tripId, request: body)
    }

    private func applyGroupsAndProcess(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let body = try req.content.decode(MockApplyGroupsAndProcessRequest.self)
        return await responseFactory.applyGroupsAndProcess(tripId: tripId, request: body)
    }

    private func review(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        return await responseFactory.review(tripId: tripId)
    }

    private func finalize(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let body = try req.content.decode(MockFinalizeTripRequest.self)
        return await responseFactory.finalize(tripId: tripId, request: body)
    }
}
