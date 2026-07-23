public struct FileToUploadDTO {
    public let clientId: String
    public let contentType: String

    public init(clientId: String, contentType: String) {
        self.clientId = clientId
        self.contentType = contentType
    }
}
