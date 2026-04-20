import Foundation

public struct SubmitBattleResultRequestDTO: Codable {
    public let winnerMediaId: String

    public init(winnerMediaId: String) {
        self.winnerMediaId = winnerMediaId
    }

    enum CodingKeys: String, CodingKey {
        case winnerMediaId = "winner_media_id"
    }
}
