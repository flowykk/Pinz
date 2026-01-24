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
