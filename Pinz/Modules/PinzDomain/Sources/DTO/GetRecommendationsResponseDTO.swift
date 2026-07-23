import Foundation

public struct GetRecommendationsResponseDTO: Codable {
    public let map: RecommendedMapDTO

    public init(map: RecommendedMapDTO) {
        self.map = map
    }
}

