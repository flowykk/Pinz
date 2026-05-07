import Foundation
import Vapor

struct TripInfoController: RouteCollection {
    let responseFactory: TripInfoResponseFactory

    func boot(routes: RoutesBuilder) throws {
        routes.get("health", use: health)

        let api = routes.grouped("api", "v1", "trips")
        api.get(":tripId", use: getTrip)
        api.patch(":tripId", use: patchTrip)
        api.patch(":tripId", "pins", ":pinId", use: patchPin)
        api.delete(":tripId", "pins", ":pinId", use: deletePin)
        api.delete(":tripId", use: deleteTrip)
        api.post(":tripId", "leave", use: leaveTrip)
    }

    private func health(_: Request) -> Response {
        let response = Response(status: .ok)
        response.body = .init(string: "ok")
        return response
    }

    private func getTrip(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        return await responseFactory.getTrip(tripId: tripId)
    }

    private func patchTrip(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let body = try req.content.decode(MockUpdateTripRequest.self)
        return await responseFactory.patchTrip(tripId: tripId, request: body)
    }

    private func patchPin(_ req: Request) async throws -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let pinId = req.parameters.get("pinId") ?? ""
        let body = try req.content.decode(MockUpdatePinRequest.self)
        return await responseFactory.patchPin(tripId: tripId, pinId: pinId, request: body)
    }

    private func deleteTrip(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        return await responseFactory.deleteTrip(tripId: tripId)
    }

    private func deletePin(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        let pinId = req.parameters.get("pinId") ?? ""
        return await responseFactory.deletePin(tripId: tripId, pinId: pinId)
    }

    private func leaveTrip(_ req: Request) async -> Response {
        let tripId = req.parameters.get("tripId") ?? ""
        return await responseFactory.leaveTrip(tripId: tripId)
    }
}
