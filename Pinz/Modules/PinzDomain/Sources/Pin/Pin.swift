import Foundation

public struct Pin: Hashable, Identifiable {
    public var id: String { name }
    
    public var name: String
    public var description: String
    public var category: PinCategory
    public var medias: [LoadedMedia]
    public var privacy: Bool
    public var startDate: Date?
    public var endDate: Date?
    public var tags: [MediaTag]

    public init(
        name: String,
        description: String,
        category: PinCategory,
        medias: [LoadedMedia],
        privacy: Bool,
        startDate: Date? = nil,
        endDate: Date? = nil,
        tags: [MediaTag]
    ) {
        self.name = name
        self.description = description
        self.category = category
        self.medias = medias
        self.privacy = privacy
        self.startDate = startDate
        self.endDate = endDate
        self.tags = tags
    }
}
