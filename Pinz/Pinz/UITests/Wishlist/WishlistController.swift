import Foundation
import Vapor

struct WishlistController: RouteCollection {
    let responseFactory: WishlistResponseFactory

    func boot(routes: RoutesBuilder) throws {
        let api = routes.grouped("api", "v1", "profile")
        api.get("desired-places", use: getPlaces)
        api.post("desired-places", use: createPlace)
        api.post("desired-places", "upload-url", use: requestUploadURL)
        api.patch("desired-places", ":placeId", use: patchPlace)
        api.delete("desired-places", ":placeId", use: deletePlace)
        api.delete("desired-places", ":placeId", "image", use: deletePlaceImage)
    }

    private func getPlaces(_: Request) async -> Response {
        await responseFactory.wishlistResponse()
    }

    private func createPlace(req: Request) async throws -> Response {
        let body = try req.content.decode(MockCreateDesiredPlaceRequest.self)
        return await responseFactory.createResponse(for: body)
    }

    private func requestUploadURL(req: Request) async throws -> Response {
        let request = try? req.content.decode(MockDesiredPlaceUploadRequest.self)
        return await responseFactory.uploadUrlResponse(request: request)
    }

    private func patchPlace(req: Request) async throws -> Response {
        let placeId = req.parameters.get("placeId") ?? ""
        let body = try req.content.decode(MockUpdateDesiredPlaceRequest.self)
        return await responseFactory.updateResponse(placeId: placeId, request: body)
    }

    private func deletePlace(req: Request) async -> Response {
        let placeId = req.parameters.get("placeId") ?? ""
        return await responseFactory.deleteResponse(placeId: placeId)
    }

    private func deletePlaceImage(req: Request) async -> Response {
        let placeId = req.parameters.get("placeId") ?? ""
        return await responseFactory.deleteImageResponse(placeId: placeId)
    }
}
