import PinzDomain

struct FeedFilterModel {
    var category: TripCategory = .none
    var season: TripSeason = .none
    var city: String = ""
    var country: String = ""
    var sortBy: FeedSortBy? = nil

    var isActive: Bool {
        category != .none || season != .none || !city.isEmpty || !country.isEmpty || sortBy != nil
    }

    var categoryParam: String? { category.apiValue }
    var seasonParam: String?   { season.apiValue }
    var cityParam: String?     { city.isEmpty ? nil : city }
    var countryParam: String?  { city.isEmpty && !country.isEmpty ? country : nil }
    var sortByParam: String?   { sortBy?.rawValue }
}

enum FeedSortBy: String, CaseIterable {
    case date   = "date"
    case rating = "rating"

    var displayName: String {
        switch self {
        case .date:   "По дате"
        case .rating: "По рейтингу"
        }
    }
}
