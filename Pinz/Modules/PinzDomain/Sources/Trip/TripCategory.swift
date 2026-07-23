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
        case .none: PinzDomainL10n.string("tripCategory.none")
        case let .custom(text): text ?? PinzDomainL10n.string("tripCategory.customOther")
        case .vacation: PinzDomainL10n.string("tripCategory.vacation")
        case .holidays: PinzDomainL10n.string("tripCategory.holidays")
        case .business: PinzDomainL10n.string("tripCategory.business")
        case .education: PinzDomainL10n.string("tripCategory.education")
        case .active: PinzDomainL10n.string("tripCategory.active")
        }
    }

    public static let allCases: [TripCategory] = [
        .custom(nil),
        .vacation,
        .holidays,
        .business,
        .education,
        .active,
    ]

    /// «Другое» — фиксированный slug `custom` в API, без произвольного текста в UI.
    public var isCustomizable: Bool { false }

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

    /// Same lower-case slugs as `apiValue` for `/recommendations` (PINZ-204).
    public var recommendationApiValue: String? { apiValue }
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
