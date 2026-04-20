import Foundation

public struct SubmitBattleResultResponseDTO: Codable {
    public let success: Bool?
    public let newBattleRating: Int?

    public init(success: Bool? = nil, newBattleRating: Int? = nil) {
        self.success = success
        self.newBattleRating = newBattleRating
    }

    enum CodingKeys: String, CodingKey {
        case success
        case newBattleRating = "new_battle_rating"
    }
}
