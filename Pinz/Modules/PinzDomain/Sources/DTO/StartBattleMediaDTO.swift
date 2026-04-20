import Foundation

public struct StartBattleMediaDTO: Codable {
    public let photoBattleMediaId: String
    public let url: String
    public let mediaType: String

    enum CodingKeys: String, CodingKey {
        case photoBattleMediaId = "media_id"
        case id
        case mediaType = "media_type"
        case type
        case url
        case mediaUrl = "media_url"
    }

    public init(photoBattleMediaId: String, mediaType: String, url: String) {
        self.photoBattleMediaId = photoBattleMediaId
        self.mediaType = mediaType
        self.url = url
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        photoBattleMediaId = Self.decodeString(container: container, keys: [.photoBattleMediaId, .id])
        url = Self.decodeString(container: container, keys: [.url, .mediaUrl])
        mediaType = Self.decodeString(container: container, keys: [.mediaType, .type])
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(photoBattleMediaId, forKey: .photoBattleMediaId)
        try container.encode(mediaType, forKey: .mediaType)
        try container.encode(url, forKey: .url)
    }

    private static func decodeString(container: KeyedDecodingContainer<CodingKeys>, keys: [CodingKeys]) -> String {
        for key in keys {
            if let stringValue = try? container.decodeIfPresent(String.self, forKey: key), !stringValue.isEmpty {
                return stringValue
            }

            if let intValue = try? container.decodeIfPresent(Int.self, forKey: key) {
                return String(intValue)
            }
        }
        return ""
    }
}
