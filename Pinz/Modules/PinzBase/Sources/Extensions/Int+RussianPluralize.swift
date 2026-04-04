import Foundation

extension Int {
    /// Возвращает правильное окончание слова "день" в зависимости от числа
    /// - Returns: "день", "дня" или "дней"
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

    /// Возвращает строку с числом и правильным окончанием
    /// - Returns: "1 день", "2 дня", "5 дней" и т.д.
    public var daysText: String {
        "\(self) \(pluralizeDays())"
    }
}
