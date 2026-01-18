public struct Pin: Hashable, Identifiable {
    public var id: String { name }
    
    public var name: String
    public var category: String
    public var medias: [LoadedMedia]
    public var privacy: Bool
    public var tags: [MediaTag]?

    public init(name: String, category: String, medias: [LoadedMedia], privacy: Bool, tags: [MediaTag]? = nil) {
        self.name = name
        self.category = category
        self.medias = medias
        self.privacy = privacy
        self.tags = tags
    }
}
