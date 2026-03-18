import Foundation

public struct PasskeyOptionsResponse: Codable {
    public let optionsJson: String

    public init(optionsJson: String) {
        self.optionsJson = optionsJson
    }

    enum CodingKeys: String, CodingKey {
        case optionsJson = "options_json"
    }
}
