public struct CreateTripDTO: Codable {
    public let tripId: String
    public let status: String
    public let uploadUrls: [UploadURLDTO]

    public init(tripId: String, status: String, uploadUrls: [UploadURLDTO]) {
        self.tripId = tripId
        self.status = status
        self.uploadUrls = uploadUrls
    }

    enum CodingKeys: String, CodingKey {
        case tripId = "trip_id"
        case status
        case uploadUrls = "upload_urls"
    }
}
