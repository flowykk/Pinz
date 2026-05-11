public enum PinCategory: PickerItem {
    case custom(String? = nil)
    case sight
    case nature
    case leisure
    case housing
    case food
    case shopping
    case transport
    case entertainment
    case event
    case sport
    case work

    public var id: Self { self }

    public var content: PickerItemContent {
        .text(self.value)
    }

    public var value: String {
        switch self {
        case let .custom(text): text ?? "Другое"
        case .sight: "Достопримечательности"
        case .nature: "Природа"
        case .leisure: "Отдых"
        case .housing: "Жилье"
        case .food: "Еда и напитки"
        case .shopping: "Шопинг"
        case .transport: "Транспорт"
        case .entertainment: "Развлечение"
        case .event: "Мероприятие"
        case .sport: "Спорт"
        case .work: "Рабочее место"
        }
    }

    public var apiValue: String {
        switch self {
        case let .custom(text):
            text?.lowercased() ?? "custom"
        case .sight: "sight"
        case .nature: "nature"
        case .leisure: "leisure"
        case .housing: "housing"
        case .food: "food"
        case .shopping: "shopping"
        case .transport: "transport"
        case .entertainment: "entertainment"
        case .event: "event"
        case .sport: "sport"
        case .work: "work"
        }
    }

    public static let allCases: [PinCategory] = [
        .custom("Другое"),
        .sight,
        .nature,
        .leisure,
        .housing,
        .food,
        .shopping,
        .transport,
        .entertainment,
        .event,
        .sport,
        .work
    ]

    public var isCustomizable: Bool {
        switch self {
        case .custom: return true
        default: return false
        }
    }
}

extension String {
    public func toPinCategory() -> PinCategory {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return .custom(nil) }
        let key = trimmed.lowercased()
        switch key {
        case "sight": return .sight
        case "nature": return .nature
        case "leisure": return .leisure
        case "housing": return .housing
        case "food": return .food
        case "shopping": return .shopping
        case "transport": return .transport
        case "entertainment": return .entertainment
        case "event": return .event
        case "sport": return .sport
        case "work": return .work
        case "custom": return .custom(nil)
        case "достопримечательность", "достопримечательности": return .sight
        case "природа": return .nature
        case "отдых": return .leisure
        case "жилье": return .housing
        case "еда и напитки": return .food
        case "шопинг": return .shopping
        case "транспорт": return .transport
        case "развлечение": return .entertainment
        case "мероприятие": return .event
        case "спорт": return .sport
        case "рабочее место": return .work
        case "другое": return .custom(nil)
        default: return .custom(trimmed)
        }
    }
}
