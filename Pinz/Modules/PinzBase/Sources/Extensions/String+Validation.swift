import Foundation

public extension String {
    var isValidEmail: Bool {
        range(of: #"^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$"#, options: .regularExpression) != nil
    }

    var isValidWishlistPlaceName: Bool {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        return trimmed.range(of: #"^[a-zA-Zа-яА-ЯёЁ0-9 ]+$"#, options: .regularExpression) != nil
    }
}
