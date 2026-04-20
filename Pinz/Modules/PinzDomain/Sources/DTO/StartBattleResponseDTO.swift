import Foundation

public struct StartBattleResponseDTO: Codable {
    public let battleId: String
    public let media: [StartBattleMediaDTO]

    enum CodingKeys: String, CodingKey {
        case battleId = "battle_id"
        case battleIdAlt = "battleId"
        case media
    }

    public init(battleId: String, media: [StartBattleMediaDTO]) {
        self.battleId = battleId
        self.media = media
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        battleId = container.decodeString(for: [.battleId, .battleIdAlt], defaultValue: "")
        media = (try? container.decode([StartBattleMediaDTO].self, forKey: .media)) ?? []
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(battleId, forKey: .battleId)
        try container.encode(media, forKey: .media)
    }
}

private extension KeyedDecodingContainer where K == StartBattleResponseDTO.CodingKeys {
    func decodeString(for keys: [K], defaultValue: String = "") -> String {
        for key in keys {
            if let stringValue = try? decodeIfPresent(String.self, forKey: key), !stringValue.isEmpty {
                return stringValue
            }

            if let intValue = try? decodeIfPresent(Int.self, forKey: key) {
                return String(intValue)
            }
        }
        return defaultValue
    }
}
