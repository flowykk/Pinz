import Foundation

public enum CensorshipField: Sendable {
    case tripName
    case tripDescription
    case pinName
    case pinDescription
}

public enum CensorshipStubs {

    public static func stub(
        for field: CensorshipField,
        entityId: String,
        locale: Locale = .current
    ) -> String {
        let pool = pool(for: field, locale: locale)
        guard !pool.isEmpty else { return "" }
        let index = Int(stableHash(entityId) % UInt64(pool.count))
        return pool[index]
    }

    private static func pool(for field: CensorshipField, locale: Locale) -> [String] {
        let code = locale.language.languageCode?.identifier ?? "en"
        let table = code == "ru" ? ruPool : enPool
        return table[field] ?? []
    }

    private static func stableHash(_ string: String) -> UInt64 {
        var hash: UInt64 = 5381
        for byte in string.utf8 {
            hash = (hash &* 33) &+ UInt64(byte)
        }
        return hash
    }

    private static let ruPool: [CensorshipField: [String]] = [
        .tripName: [
            "Крутое путешествие",
            "Лучшие выходные",
            "Мой маршрут",
            "Прогулка по городу",
            "Удивительный день",
            "Заметки из поездки",
            "История одного трипа"
        ],
        .tripDescription: [
            "Тут скоро появится описание",
            "Маршрут в работе",
            "Эта поездка пока без подробностей",
            "Описание готовится",
            "Подробности добавим позже"
        ],
        .pinName: [
            "Точка маршрута",
            "Интересное место",
            "Локация",
            "Атмосферное место",
            "Любимая точка"
        ],
        .pinDescription: [
            "Без описания",
            "Здесь будет история",
            "Заметки скоро появятся",
            "Описание готовится",
            "Подробности позже"
        ]
    ]

    private static let enPool: [CensorshipField: [String]] = [
        .tripName: [
            "Cool trip",
            "My lovely trip",
            "Weekend adventure",
            "City walk",
            "Travel notes",
            "Amazing day",
            "Memories on the road"
        ],
        .tripDescription: [
            "Description coming soon",
            "Trip details in progress",
            "No details yet",
            "Trip story to follow",
            "Details will be added later"
        ],
        .pinName: [
            "A nice spot",
            "Point on the map",
            "Favorite place",
            "A lovely location",
            "Memorable spot"
        ],
        .pinDescription: [
            "No description",
            "Story to be added",
            "Notes coming soon",
            "Description in progress",
            "Details later"
        ]
    ]
}
