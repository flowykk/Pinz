public enum TripSeason: PickerItem {
    case none
    case summer
    case autumn
    case winter
    case spring

    public var id: Self { self }

    public var content: PickerItemContent {
        switch self {
        case .none: .text("Не выбрано")
        case .summer: .text("Лето")
        case .autumn: .text("Осень")
        case .winter: .text("Зима")
        case .spring: .text("Весна")
        }
    }

    public var value: String {
        switch content {
        case let .text(text): text
        case let .icon(icon, _): icon
        }
    }

    public static let allCases: [TripSeason] = [
        .summer,
        .autumn,
        .winter,
        .spring,
    ]

    public var isCustomizable: Bool { false }

    public var apiValue: String? {
        switch self {
        case .none:   return nil
        case .summer: return "summer"
        case .autumn: return "autumn"
        case .winter: return "winter"
        case .spring: return "spring"
        }
    }

    /// Cyrillic value expected by `/recommendations` endpoints per backend guide.
    public var recommendationApiValue: String? {
        switch self {
        case .none:   return nil
        case .summer: return "Лето"
        case .autumn: return "Осень"
        case .winter: return "Зима"
        case .spring: return "Весна"
        }
    }
}
