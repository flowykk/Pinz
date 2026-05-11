import Foundation

public extension TripDTO {
    func toTrip() -> Trip {
        let season: TripSeason = {
            guard let raw = self.season?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
                return .none
            }
            let key = raw.lowercased()
            switch key {
            case "summer": return .summer
            case "autumn", "fall": return .autumn
            case "winter": return .winter
            case "spring": return .spring
            case "лето": return .summer
            case "осень": return .autumn
            case "зима": return .winter
            case "весна": return .spring
            default: return .none
            }
        }()

        let category: TripCategory = {
            guard let raw = self.category?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
                return .none
            }
            let key = raw.lowercased()
            switch key {
            case "vacation": return .vacation
            case "holidays", "holiday": return .holidays
            case "business": return .business
            case "education": return .education
            case "active": return .active
            case "custom": return .custom(nil)
            case "отпуск": return .vacation
            case "командировка": return .business
            case "выходные": return .holidays
            case "активный отдых": return .active
            case "образование": return .education
            case "другое": return .custom(nil)
            default: return .custom(raw)
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
