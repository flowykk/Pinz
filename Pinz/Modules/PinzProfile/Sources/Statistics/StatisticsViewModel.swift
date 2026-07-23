import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class StatisticsViewModel {
    private enum VisitedLocationType {
        static let country = "country"
        static let city = "city"
    }

    enum Route {
        case back
    }

    enum Intent {
        case loadStats
        case navigate(Route)
    }

    var isLoading = false

    private(set) var totalTrips = 0
    private(set) var totalPins = 0
    private(set) var totalMedia = 0
    private(set) var totalLikes = 0
    private(set) var totalDislikes = 0
    private(set) var battlesFinished = 0
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

        totalTrips = 0
        totalPins = 0
        totalMedia = 0
        totalLikes = 0
        totalDislikes = 0
        battlesFinished = 0
        visitedCountries = []
        visitedCities = []

        do {
            let response = try await networkService.getProfileStats()
            totalTrips = response.totalTrips ?? 0
            totalPins = response.totalPins ?? 0
            totalMedia = response.totalMedia ?? 0
            totalLikes = response.totalLikes ?? 0
            totalDislikes = response.totalDislikes ?? 0
            battlesFinished = response.battlesFinished ?? 0
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
