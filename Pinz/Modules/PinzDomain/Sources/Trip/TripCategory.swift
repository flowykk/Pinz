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

    public var apiValue: String? {
        switch self {
        case .none:       return nil
        case .vacation:   return "vacation"
        case .holidays:   return "holidays"
        case .business:   return "business"
        case .education:  return "education"
        case .active:     return "active"
        case .custom:     return "custom"
        }
    }
}

extension TripCategory: Equatable {
    public static func == (lhs: TripCategory, rhs: TripCategory) -> Bool {
        switch (lhs, rhs) {
        case (.none, .none), (.vacation, .vacation), (.holidays, .holidays),
             (.business, .business), (.education, .education), (.active, .active):
            return true
        case (.custom(let a), .custom(let b)):
            return a == b
        default:
            return false
        }
    }
}
