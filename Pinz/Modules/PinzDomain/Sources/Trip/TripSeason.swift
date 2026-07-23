public enum TripSeason: PickerItem {
    case none
    case summer
    case autumn
    case winter
    case spring

    public var id: Self { self }

    public var content: PickerItemContent {
        switch self {
        case .none: .text(PinzDomainL10n.string("tripSeason.none"))
        case .summer: .text(PinzDomainL10n.string("tripSeason.summer"))
        case .autumn: .text(PinzDomainL10n.string("tripSeason.autumn"))
        case .winter: .text(PinzDomainL10n.string("tripSeason.winter"))
        case .spring: .text(PinzDomainL10n.string("tripSeason.spring"))
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

    /// Same lower-case slugs as `apiValue` for `/recommendations` (PINZ-204).
    public var recommendationApiValue: String? { apiValue }
}
