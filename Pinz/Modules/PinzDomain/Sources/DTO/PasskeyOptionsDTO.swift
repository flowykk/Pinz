import Foundation

public struct PasskeyOptionsDTO: Codable {
    public let optionsJson: String

    public init(optionsJson: String) {
        self.optionsJson = optionsJson
    }

    enum CodingKeys: String, CodingKey {
        case optionsJson = "options_json"
    }
}
