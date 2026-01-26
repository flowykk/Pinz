public enum TripCategory: PickerItem {
    case none
    case vacation
    case holidays
    case business
    case education
    case active

    public var id: Self { self }

    public var content: PickerItemContent {
        switch self {
        case .none: .text("Не выбрано")
        case .vacation: .text("Отпуск")
        case .holidays: .text("Выходные")
        case .business: .text("Командировка")
        case .education: .text("Образование")
        case .active:  .text("Активный отдых")
        }
    }

    public var value: String {
        switch content {
        case let .text(text): text
        case let .icon(icon, _): icon
        }
    }

    public static let allCases: [TripCategory] = [
        .none,
        .vacation,
        .holidays,
        .business,
        .education,
        .active,
    ]

    public var isCustomizable: Bool { false }
}
