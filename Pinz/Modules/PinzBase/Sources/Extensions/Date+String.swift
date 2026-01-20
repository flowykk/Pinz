import Foundation

extension Date {
    public init?(fromDateString string: String) {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "ru_RU_POSIX")
        formatter.dateFormat = "dd.MM.yyyy"
        if let date = formatter.date(from: string) {
            self = date
        } else {
            return nil
        }
    }

    public var formattedToDayMonthYear: String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "ru_RU_POSIX")
        formatter.dateFormat = "dd.MM.yyyy"
        return formatter.string(from: self)
    }
}
