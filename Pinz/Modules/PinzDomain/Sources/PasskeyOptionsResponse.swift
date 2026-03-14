import Foundation

public struct PasskeyOptionsResponse: Codable {
    public let optionsJson: String

    enum CodingKeys: String, CodingKey {
        case optionsJson = "options_json"
    }
}
