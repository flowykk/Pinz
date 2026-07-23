import Foundation

public struct VisitedLocationDTO: Codable {
    public let name: String?
    public let lastVisitedAtUnix: Int?
    public let visitsCount: Int?

    public init(
        name: String? = nil,
        lastVisitedAtUnix: Int? = nil,
        visitsCount: Int? = nil
    ) {
        self.name = name
        self.lastVisitedAtUnix = lastVisitedAtUnix
        self.visitsCount = visitsCount
    }

    enum CodingKeys: String, CodingKey {
        case name
        case lastVisitedAtUnix = "last_visited_at_unix"
        case visitsCount = "visits_count"
    }
}
