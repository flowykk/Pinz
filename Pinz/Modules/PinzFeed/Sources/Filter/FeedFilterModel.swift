import PinzBase
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

    /// Normalized token for API (PINZ-216): trim + lower-case.
    private func normalizedToken(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    var cityParam: String? {
        let token = normalizedToken(city)
        return token.isEmpty ? nil : token
    }

    var countryParam: String? {
        guard cityParam == nil else { return nil }
        let token = normalizedToken(country)
        return token.isEmpty ? nil : token
    }
    var sortByParam: String?   { sortBy?.rawValue }

    var recommendationCategoryParam: String? { category.recommendationApiValue }
    var recommendationSeasonParam: String?   { season.recommendationApiValue }
}

enum FeedSortBy: String, CaseIterable {
    case date   = "date"
    case rating = "rating"

    var displayName: String {
        switch self {
        case .date:   PinzBaseStrings.Feed.Filter.Sort.date
        case .rating: PinzBaseStrings.Feed.Filter.Sort.rating
        }
    }
}
