import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class StatisticsViewModel {
    private enum VisitedLocationType {
        static let country = "Country"
        static let city = "City"
    }

    enum Route {
        case back
    }

    enum Intent {
        case loadStats
        case navigate(Route)
    }

    var isLoading = false

    private(set) var tripsCount = 0
    private(set) var pinsCount = 0
    private(set) var mediaCount = 0
    private(set) var likesCount = 0
    private(set) var dislikesCount = 0
    private(set) var battlesCount = 0
    private(set) var visitedCountries: [VisitedLocationDTO] = []
    private(set) var visitedCities: [VisitedLocationDTO] = []

    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?

    init(networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .loadStats:
            guard !isLoading else {
                return
            }

            isLoading = true
            Task {
                await loadStats()
            }
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func loadStats() async {
        defer {
            isLoading = false
        }

        tripsCount = 0
        pinsCount = 0
        mediaCount = 0
        likesCount = 0
        dislikesCount = 0
        battlesCount = 0
        visitedCountries = []
        visitedCities = []

        do {
            let response = try await networkService.getProfileStats()
            tripsCount = response.tripsCount ?? 0
            pinsCount = response.pinsCount ?? 0
            mediaCount = response.mediaCount ?? 0
            likesCount = response.likesCount ?? 0
            dislikesCount = response.dislikesCount ?? 0
            battlesCount = response.battlesCount ?? 0
        } catch {
            print("[Statistics] Failed to load stats: \(error)")
        }

        do {
            visitedCountries = try await networkService.getVisitedLocations(type: VisitedLocationType.country).locations
        } catch {
            print("[Statistics] Failed to load visited countries: \(error)")
            visitedCountries = []
        }

        do {
            visitedCities = try await networkService.getVisitedLocations(type: VisitedLocationType.city).locations
        } catch {
            print("[Statistics] Failed to load visited cities: \(error)")
            visitedCities = []
        }
    }
}
