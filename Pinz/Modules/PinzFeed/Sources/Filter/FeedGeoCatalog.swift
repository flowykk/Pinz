import Foundation

enum FeedGeoCatalog {

    struct GeoLocalized: Codable, Hashable, Sendable {
        let ru: String
        let eng: String

        fileprivate func display(preferRussian: Bool) -> String {
            preferRussian ? ru : eng
        }
    }

    struct Entry: Hashable, Identifiable, Sendable {
        let key: String
        let localized: GeoLocalized

        var id: String { key }
    }

    private static let bundle = Bundle.module
    private static let lock = NSLock()

    private static var countriesEntries: [Entry]?
    private static var countryByKey: [String: GeoLocalized]?
    private static var countryLowerToCanonical: [String: String]?

    private static var citiesEntries: [Entry]?
    private static var cityByKey: [String: GeoLocalized]?
    private static var cityLowerToCanonical: [String: String]?

    private static var preferRussian: Bool {
        Locale.current.language.languageCode?.identifier == "ru"
    }

    static func displayLabel(for entry: Entry) -> String {
        entry.localized.display(preferRussian: preferRussian)
    }

    static func allCountries() -> [Entry] {
        lock.lock()
        defer { lock.unlock() }
        if let countriesEntries { return countriesEntries }
        let raw = decodeCatalog(resource: "countries")
        countryByKey = raw
        countryLowerToCanonical = buildLowerToCanonical(from: raw)
        let pr = Self.preferRussian
        let entries = raw
            .map { Entry(key: $0.key, localized: $0.value) }
            .sorted { lhs, rhs in
                let l = lhs.localized.display(preferRussian: pr)
                let r = rhs.localized.display(preferRussian: pr)
                return l.localizedStandardCompare(r) == .orderedAscending
            }
        countriesEntries = entries
        return entries
    }

    static func allCities() -> [Entry] {
        lock.lock()
        defer { lock.unlock() }
        if let citiesEntries { return citiesEntries }
        let raw = decodeCatalog(resource: "cities")
        cityByKey = raw
        cityLowerToCanonical = buildLowerToCanonical(from: raw)
        let pr = Self.preferRussian
        let entries = raw
            .map { Entry(key: $0.key, localized: $0.value) }
            .sorted { lhs, rhs in
                let l = lhs.localized.display(preferRussian: pr)
                let r = rhs.localized.display(preferRussian: pr)
                return l.localizedStandardCompare(r) == .orderedAscending
            }
        citiesEntries = entries
        return entries
    }

    static func countryDisplay(forSlug slug: String) -> String {
        let trimmed = slug.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "" }
        lock.lock()
        let needsLoad = countryByKey == nil
        lock.unlock()
        if needsLoad { _ = allCountries() }
        lock.lock()
        defer { lock.unlock() }
        let pr = Self.preferRussian
        if let loc = lookupLocalized(trimmed, map: countryByKey, lowerToCanon: countryLowerToCanonical) {
            return loc.display(preferRussian: pr)
        }
        return fallbackDisplay(forKey: trimmed.lowercased())
    }

    static func cityDisplay(forSlug slug: String) -> String {
        let trimmed = slug.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "" }
        lock.lock()
        let needsLoad = cityByKey == nil
        lock.unlock()
        if needsLoad { _ = allCities() }
        lock.lock()
        defer { lock.unlock() }
        let pr = Self.preferRussian
        if let loc = lookupLocalized(trimmed, map: cityByKey, lowerToCanon: cityLowerToCanonical) {
            return loc.display(preferRussian: pr)
        }
        return fallbackDisplay(forKey: trimmed.lowercased())
    }

    /// `eng` from the bundled catalog for a lower-case `region_name` from recommendations/feed API.
    static func englishDisplay(forRegionSlug slug: String, regionType: String?) -> String {
        let s = slug.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !s.isEmpty else { return slug }
        if let loc = geoLocalized(forRegionSlug: s, regionType: regionType) {
            return loc.eng
        }
        return fallbackDisplay(forKey: s.lowercased())
    }

    private static func geoLocalized(forRegionSlug s: String, regionType: String?) -> GeoLocalized? {
        let t = (regionType ?? "").lowercased()
        func countryLoc() -> GeoLocalized? {
            _ = allCountries()
            lock.lock()
            defer { lock.unlock() }
            return lookupLocalized(s, map: countryByKey, lowerToCanon: countryLowerToCanonical)
        }
        func cityLoc() -> GeoLocalized? {
            _ = allCities()
            lock.lock()
            defer { lock.unlock() }
            return lookupLocalized(s, map: cityByKey, lowerToCanon: cityLowerToCanonical)
        }
        switch t {
        case "country":
            return countryLoc() ?? cityLoc()
        case "city":
            return cityLoc() ?? countryLoc()
        default:
            return cityLoc() ?? countryLoc()
        }
    }

    private static func lookupLocalized(
        _ slug: String,
        map: [String: GeoLocalized]?,
        lowerToCanon: [String: String]?
    ) -> GeoLocalized? {
        guard let map, let lowerToCanon else { return nil }
        let lower = slug.lowercased()
        if let canonical = lowerToCanon[lower], let loc = map[canonical] {
            return loc
        }
        return map[slug]
    }

    private static func buildLowerToCanonical(from raw: [String: GeoLocalized]) -> [String: String] {
        var out: [String: String] = [:]
        out.reserveCapacity(raw.count)
        for key in raw.keys {
            out[key.lowercased()] = key
        }
        return out
    }

    private static func decodeCatalog(resource: String) -> [String: GeoLocalized] {
        guard let url = bundle.url(forResource: resource, withExtension: "json"),
              let data = try? Data(contentsOf: url),
              let dict = try? JSONDecoder().decode([String: GeoLocalized].self, from: data)
        else {
            return [:]
        }
        return dict
    }

    private static func fallbackDisplay(forKey key: String) -> String {
        key.replacingOccurrences(of: "-", with: " ").localizedCapitalized
    }
}
