import Foundation

public extension TripDTO {
    func toTrip() -> Trip {
        let season: TripSeason = {
            switch self.season?.lowercased() {
            case "summer": return .summer
            case "autumn", "fall": return .autumn
            case "winter": return .winter
            case "spring": return .spring
            default: return .none
            }
        }()

        let category: TripCategory = {
            switch self.category?.lowercased() {
            case "vacation": return .vacation
            case "holidays", "holiday": return .holidays
            case "business": return .business
            case "education": return .education
            case "active": return .active
            default: return self.category.map { .custom($0) } ?? .none
            }
        }()

        return Trip(
            id: id,
            name: name,
            description: description,
            pins: [],
            season: season,
            startDate: startDateUnix.map { Date(timeIntervalSince1970: Double($0)) },
            endDate: endDateUnix.map { Date(timeIntervalSince1970: Double($0)) },
            category: category,
            participantsCount: participantsCount ?? 0,
            mediaCount: mediaCount ?? 0,
            coverUrl: coverUrl,
            ownerUserId: ownerUserId,
            privacyLevel: privacyLevel,
            status: status,
            isPublished: isPublished,
            isGenerated: isGenerated,
            likesCount: likesCount,
            dislikesCount: dislikesCount,
            createdAt: Date(timeIntervalSince1970: Double(createdAtUnix)),
            updatedAt: Date(timeIntervalSince1970: Double(updatedAtUnix))
        )
    }
}
