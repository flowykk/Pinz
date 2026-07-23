import Foundation

extension Int {

    public func pluralizeDays() -> String {
        let lastDigit = self % 10
        let lastTwoDigits = self % 100

        if lastDigit == 1 && lastTwoDigits != 11 {
            return "день"
        } else if (2...4).contains(lastDigit) && !(12...14).contains(lastTwoDigits) {
            return "дня"
        } else {
            return "дней"
        }
    }

    public var daysText: String {
        "\(self) \(pluralizeDays())"
    }
}
