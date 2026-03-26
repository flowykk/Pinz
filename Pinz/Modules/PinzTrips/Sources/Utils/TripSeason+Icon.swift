import PinzDomain
import PinzUI

enum TripSeasonIcon: String, Setting.Icon {
    case summer = "sun.max.fill"
    case autumn = "cloud.fill"
    case winter = "snowflake"
    case spring = "leaf.fill"
}

extension TripSeason {
    static func icon(for season: Self) -> TripSeasonIcon {
        switch season {
        case .none: return .summer
        case .summer: return .summer
        case .autumn: return .autumn
        case .winter: return .winter
        case .spring: return .spring
        }
    }
}
