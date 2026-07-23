import Foundation

public struct DesiredPlace: Hashable, Identifiable {
    public var id: String
    public var name: String
    public var description: String
    public var imageUrl: String?
    public let createdAt: Int

    public init(id: String, name: String, description: String, imageUrl: String? = nil, createdAt: Int = Int(Date().timeIntervalSince1970)) {
        self.id = id
        self.name = name
        self.description = description
        self.imageUrl = imageUrl
        self.createdAt = createdAt
    }
}
