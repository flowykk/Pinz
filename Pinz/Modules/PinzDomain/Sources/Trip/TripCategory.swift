public enum TripCategory: PickerItem {
    case none
    case custom(String? = nil)
    case vacation
    case holidays
    case business
    case education
    case active

    public var id: Self { self }

    public var content: PickerItemContent {
        .text(self.value)
    }

    public var value: String {
        switch self {
        case .none: "Не выбрано"
        case let .custom(text): text ?? "Другое"
        case .vacation: "Отпуск"
        case .holidays: "Выходные"
        case .business: "Командировка"
        case .education: "Образование"
        case .active: "Активный отдых"
        }
    }

    public static let allCases: [TripCategory] = [
        .custom("Другое"),
        .vacation,
        .holidays,
        .business,
        .education,
        .active,
    ]

    public var isCustomizable: Bool {
        switch self {
        case .custom: return true
        default: return false
        }
    }
}
