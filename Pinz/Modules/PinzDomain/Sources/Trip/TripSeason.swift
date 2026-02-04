public enum TripSeason: PickerItem {
    case summer
    case autumn
    case winter
    case spring

    public var id: Self { self }

    public var content: PickerItemContent {
        switch self {
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
}
