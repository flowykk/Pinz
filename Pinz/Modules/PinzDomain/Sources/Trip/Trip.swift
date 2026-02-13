import Foundation
import SwiftUI

public struct Trip: Hashable {
    public var name: String
    public var image: UIImage?
    public var description: String?
    public var pins: [Pin]
    public var season: TripSeason
    public var startDate: Date?
    public var endDate: Date?
    public var category: TripCategory
    public var members: [TripMember]

    public init(
        name: String,
        image: UIImage? = nil,
        description: String? = nil,
        pins: [Pin],
        season: TripSeason,
        startDate: Date? = nil,
        endDate: Date? = nil,
        category: TripCategory,
        members: [TripMember] = []
    ) {
        self.name = name
        self.image = image
        self.description = description
        self.pins = pins
        self.season = season
        self.startDate = startDate
        self.endDate = endDate
        self.category = category
        self.members = members
    }
}

extension Trip {
    public static func stub() -> Trip {
        let description = "В Сочи мы выбрались прошлым летом. Решили туда съездить, так как никогда раньше в этом городе- курорте не были, а это все же довольно популярное место. Многие в восторге от Сочи, город дорогой, популярный и престижный. Часто туда переезжают жить, ну или планируют это сделать в ближайшее время. Словом, город так и манит всех в свои теплые объятия, прямо как Москва. Ну вот и мы собрались на летний отдых туда, чтобы посмотреть, правда ли там так здорово. Для поездки выбрали месяц август. Да, это самый пик тур сезона, но вот так получилось. В Сочи ехали из Нижнего на поезде. На самом деле удобно, что без пересадок и до самого вокзала. Правда ехать пару суток, но я люблю поезда, поэтому даже в плацкарте на этот раз добрались без особых проблем. В поезде был вагон ресторан, и мы несколько раз заказывали еду из него."


        return Trip(
            name: "Нижний Новгород",
            image: PinzDomainAsset.defaultPlaceholder.image,
            description: description,
            pins: Pin.stubs(),
            season: .summer,
            startDate: Date(fromDateString: "03.01.2026"),
            endDate: Date(fromDateString: "09.01.2026"),
            category: .vacation,
            members: TripMember.stubs()
        )
    }
}

extension Date {
    public init?(fromDateString string: String) {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "ru_RU_POSIX")
        formatter.dateFormat = "dd.MM.yyyy"
        if let date = formatter.date(from: string) {
            self = date
        } else {
            return nil
        }
    }

    public var formattedToDayMonthYear: String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "ru_RU_POSIX")
        formatter.dateFormat = "dd.MM.yyyy"
        return formatter.string(from: self)
    }
}
