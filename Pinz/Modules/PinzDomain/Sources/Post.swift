import Foundation

public struct Post: Equatable, Identifiable, Hashable {
    public var id: String
    public var name: String
    public var description: String?
    public var participants: Int
    public var likes: Int
    public var dislikes: Int
    public var favorites: Int
    public var views: Int
    public var pins: [Pin]

    public init(
        id: String,
        name: String,
        description: String? = nil,
        participants: Int,
        likes: Int,
        dislikes: Int,
        favorites: Int,
        views: Int,
        pins: [Pin]
    ) {
        self.id = id
        self.name = name
        self.description = description
        self.participants = participants
        self.likes = likes
        self.dislikes = dislikes
        self.favorites = favorites
        self.views = views
        self.pins = pins
    }
}

extension Post {
    public static var stub: Post {
        Post(
            id: UUID().uuidString,
            name: "Paris Trip",
            description: "Amazing week in Paris",
            participants: 12,
            likes: 234,
            dislikes: 21,
            favorites: 45,
            views: 1200,
            pins: Pin.stubs()
        )
    }
}
