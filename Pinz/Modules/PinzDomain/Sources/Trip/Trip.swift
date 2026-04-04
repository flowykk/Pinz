import Foundation
import SwiftUI

public struct Trip: Hashable, Identifiable {
    public let id: String
    public var name: String
    public var image: UIImage?
    public var description: String?
    public var pins: [Pin]
    public var season: TripSeason
    public var startDate: Date?
    public var endDate: Date?
    public var category: TripCategory
    public var members: [TripMember]
    public var coverUrl: String?
    public var ownerUserId: String
    public var privacyLevel: String?
    public var status: String?
    public var isPublished: Bool
    public var isGenerated: Bool
    public var likesCount: Int
    public var dislikesCount: Int
    public var createdAt: Date
    public var updatedAt: Date

    public init(
        id: String = UUID().uuidString,
        name: String,
        image: UIImage? = nil,
        description: String? = nil,
        pins: [Pin],
        season: TripSeason,
        startDate: Date? = nil,
        endDate: Date? = nil,
        category: TripCategory,
        members: [TripMember] = [],
        coverUrl: String? = nil,
        ownerUserId: String = "",
        privacyLevel: String? = nil,
        status: String? = nil,
        isPublished: Bool = false,
        isGenerated: Bool = false,
        likesCount: Int = 0,
        dislikesCount: Int = 0,
        createdAt: Date = Date(),
        updatedAt: Date = Date()
    ) {
        self.id = id
        self.name = name
        self.image = image
        self.description = description
        self.pins = pins
        self.season = season
        self.startDate = startDate
        self.endDate = endDate
        self.category = category
        self.members = members
        self.coverUrl = coverUrl
        self.ownerUserId = ownerUserId
        self.privacyLevel = privacyLevel
        self.status = status
        self.isPublished = isPublished
        self.isGenerated = isGenerated
        self.likesCount = likesCount
        self.dislikesCount = dislikesCount
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

extension Trip {
    public static func stub() -> Trip {
        let description = "В Сочи мы выбрались прошлым летом. Решили туда съездить, так как никогда раньше в этом городе- курорте не были, а это все же довольно популярное место. Многие в восторге от Сочи, город дорогой, популярный и престижный. Часто туда переезжают жить, ну или планируют это сделать в ближайшее время. Словом, город так и манит всех в свои теплые объятия, прямо как Москва. Ну вот и мы собрались на летний отдых туда, чтобы посмотреть, правда ли там так здорово. Для поездки выбрали месяц август. Да, это самый пик тур сезона, но вот так получилось. В Сочи ехали из Нижнего на поезде. На самом деле удобно, что без пересадок и до самого вокзала. Правда ехать пару суток, но я люблю поезда, поэтому даже в плацкарте на этот раз добрались без особых проблем. В поезде был вагон ресторан, и мы несколько раз заказывали еду из него."

        return Trip(
            id: "trip-default-nn",
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
    
    public static func stubs() -> [Trip] {
        let sochi = Trip(
            id: "trip-sochi",
            name: "Сочи",
            image: PinzDomainAsset.groupPlaceholder.image,
            description: "Летний отдых на море с посещением всех главных достопримечательностей города",
            pins: Array(Pin.stubs().prefix(3)),
            season: .summer,
            startDate: Date(fromDateString: "15.06.2026"),
            endDate: Date(fromDateString: "25.06.2026"),
            category: .vacation,
            members: Array(TripMember.stubs().prefix(2))
        )
        
        return [stub(), sochi]
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
