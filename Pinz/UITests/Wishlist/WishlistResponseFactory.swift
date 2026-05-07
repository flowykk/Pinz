import Foundation
import Vapor

struct MockWishlistPlaceSnapshot {
    let id: String
    let name: String
    let description: String
    let imageUrl: String?
}

struct MockWishlistPlaceDTO: Content {
    let id: String
    let name: String
    let description: String
    let imageUrl: String?
    let createdAt: Int

    enum CodingKeys: String, CodingKey {
        case id, name, description
        case imageUrl = "image_url"
        case createdAt = "created_at"
    }
}

struct MockWishlistListResponse: Content {
    let places: [MockWishlistPlaceDTO]
}

struct MockDesiredPlaceUploadRequest: Content {
    let filename: String
    let contentType: String
}

struct MockDesiredPlaceUploadResponse: Content {
    let uploadUrl: String
    let s3Key: String
    let expiresAtUnix: Int

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case s3Key = "s3_key"
        case expiresAtUnix = "expires_at_unix"
    }
}

struct MockCreateDesiredPlaceRequest: Content {
    let name: String
    let description: String
    let s3Key: String?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case s3Key = "s3_key"
    }
}

struct MockUpdateDesiredPlaceRequest: Content {
    let name: String
    let description: String
    let imageS3Key: String?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case imageS3Key = "image_s3_key"
    }
}

struct MockSuccessResponse: Content {
    let success: Bool
}

actor MockWishlistState {
    private let mockedUploadUrl = "http://localhost:8080/desires/upload-url"
    private let mockedS3Key = "wishlist/stub/image.jpg"

    private var places: [MockWishlistPlaceSnapshot]
    private(set) var getCount = 0
    private(set) var createCount = 0
    private(set) var updateCount = 0
    private(set) var deleteCount = 0
    private(set) var uploadRequestCount = 0
    private(set) var deleteImageCount = 0
    private(set) var lastUploadFilename: String?
    private(set) var lastUploadContentType: String?

    init(initialPlaces: [MockWishlistPlaceSnapshot] = []) {
        self.places = initialPlaces
    }

    func getPlaces() async -> [MockWishlistPlaceSnapshot] {
        getCount += 1
        return places
    }

    func createPlace(name: String, description: String, s3Key: String?) async -> MockWishlistPlaceSnapshot {
        createCount += 1
        let created = MockWishlistPlaceSnapshot(
            id: UUID().uuidString,
            name: name,
            description: description,
            imageUrl: s3Key.flatMap { "https://cdn.pinz.test/\($0)" }
        )
        places.append(created)
        return created
    }

    func requestUpload(filename: String, contentType: String) async -> MockWishlistPlaceSnapshot? {
        uploadRequestCount += 1
        lastUploadFilename = filename
        lastUploadContentType = contentType
        return places.first
    }

    func updatePlace(id: String, request: MockUpdateDesiredPlaceRequest) async -> MockWishlistPlaceSnapshot? {
        updateCount += 1
        guard let index = places.firstIndex(where: { $0.id == id }) else {
            return nil
        }

        places[index] = MockWishlistPlaceSnapshot(
            id: places[index].id,
            name: request.name,
            description: request.description,
            imageUrl: request.imageS3Key.flatMap { _ in places[index].imageUrl }
        )
        return places[index]
    }

    func deletePlace(id: String) async -> Bool {
        deleteCount += 1
        let before = places.count
        places.removeAll { $0.id == id }
        return places.count != before
    }

    func deletePlaceImage(id: String) async -> Bool {
        deleteImageCount += 1
        guard let index = places.firstIndex(where: { $0.id == id }) else {
            return false
        }
        places[index] = MockWishlistPlaceSnapshot(
            id: places[index].id,
            name: places[index].name,
            description: places[index].description,
            imageUrl: nil
        )
        return true
    }

    func requestCounts() async -> (getCount: Int, createCount: Int, updateCount: Int, deleteCount: Int, uploadRequestCount: Int, deleteImageCount: Int) {
        (getCount, createCount, updateCount, deleteCount, uploadRequestCount, deleteImageCount)
    }

    func uploadMeta() async -> (filename: String?, contentType: String?) {
        (lastUploadFilename, lastUploadContentType)
    }

    func makeUploadResponse() -> MockDesiredPlaceUploadResponse {
        MockDesiredPlaceUploadResponse(
            uploadUrl: mockedUploadUrl,
            s3Key: mockedS3Key,
            expiresAtUnix: Int(Date().timeIntervalSince1970) + 1200
        )
    }
}

private func makeMockPlaceDTO(from snapshot: MockWishlistPlaceSnapshot, createdAt: Int) -> MockWishlistPlaceDTO {
    MockWishlistPlaceDTO(
        id: snapshot.id,
        name: snapshot.name,
        description: snapshot.description,
        imageUrl: snapshot.imageUrl,
        createdAt: createdAt
    )
}

struct WishlistResponseFactory {
    private let state: MockWishlistState
    private let createdAt = Int(Date().timeIntervalSince1970)

    init(initialWishlist: [MockWishlistPlaceSnapshot] = []) {
        self.state = MockWishlistState(initialPlaces: initialWishlist)
    }

    func wishlistResponse() async -> Response {
        let places = await state.getPlaces()
        let dto = MockWishlistListResponse(places: places.map { makeMockPlaceDTO(from: $0, createdAt: createdAt) })
        return encode(dto)
    }

    func uploadUrlResponse(request: MockDesiredPlaceUploadRequest?) async -> Response {
        if let request {
            _ = await state.requestUpload(filename: request.filename, contentType: request.contentType)
        }
        let response = await state.makeUploadResponse()
        return encode(response)
    }

    func createResponse(for request: MockCreateDesiredPlaceRequest) async -> Response {
        let created = await state.createPlace(name: request.name, description: request.description, s3Key: request.s3Key)
        return encode(makeMockPlaceDTO(from: created, createdAt: createdAt))
    }

    func updateResponse(placeId: String, request: MockUpdateDesiredPlaceRequest) async -> Response {
        guard let updated = await state.updatePlace(id: placeId, request: request) else {
            let response = Response(status: .notFound)
            return response
        }
        return encode(makeMockPlaceDTO(from: updated, createdAt: createdAt))
    }

    func deleteResponse(placeId: String) async -> Response {
        let success = await state.deletePlace(id: placeId)
        return encode(MockSuccessResponse(success: success))
    }

    func deleteImageResponse(placeId: String) async -> Response {
        let success = await state.deletePlaceImage(id: placeId)
        return encode(MockSuccessResponse(success: success))
    }

    func getCounts() async -> (get: Int, create: Int, update: Int, delete: Int, upload: Int, deleteImage: Int) {
        let counts = await state.requestCounts()
        return (counts.getCount, counts.createCount, counts.updateCount, counts.deleteCount, counts.uploadRequestCount, counts.deleteImageCount)
    }

    private func encode<T: Content>(_ value: T) -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(value)
        return response
    }

}
