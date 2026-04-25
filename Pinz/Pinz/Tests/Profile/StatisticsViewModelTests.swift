import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class StatisticsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: StatisticsViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = StatisticsViewModel(networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    // MARK: - Navigation

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    // MARK: - loadStats — success

    func test_dispatch_loadStats_success_setsAllCounts() async throws {
        mockNetwork.getProfileStatsResult = .success(
            UserStatsResponseDTO(tripsCount: 5, pinsCount: 10, mediaCount: 20, likesCount: 3, dislikesCount: 1, battlesCount: 7)
        )

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.tripsCount, 5)
        XCTAssertEqual(sut.pinsCount, 10)
        XCTAssertEqual(sut.mediaCount, 20)
        XCTAssertEqual(sut.likesCount, 3)
        XCTAssertEqual(sut.dislikesCount, 1)
        XCTAssertEqual(sut.battlesCount, 7)
    }

    func test_dispatch_loadStats_nilStatsValues_defaultsToZero() async throws {
        mockNetwork.getProfileStatsResult = .success(UserStatsResponseDTO())

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.tripsCount, 0)
        XCTAssertEqual(sut.pinsCount, 0)
        XCTAssertEqual(sut.mediaCount, 0)
        XCTAssertEqual(sut.likesCount, 0)
        XCTAssertEqual(sut.dislikesCount, 0)
        XCTAssertEqual(sut.battlesCount, 0)
    }

    func test_dispatch_loadStats_success_setsVisitedCountries() async throws {
        mockNetwork.getVisitedLocationsCountryResult = .success(
            VisitedLocationsResponseDTO(locations: [
                VisitedLocationDTO(name: "Russia", visitsCount: 2),
                VisitedLocationDTO(name: "France", visitsCount: 1)
            ])
        )
        mockNetwork.getVisitedLocationsCityResult = .success(VisitedLocationsResponseDTO(locations: []))

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.visitedCountries.count, 2)
        XCTAssertEqual(sut.visitedCountries.first?.name, "Russia")
        XCTAssertTrue(sut.visitedCities.isEmpty)
    }

    func test_dispatch_loadStats_success_setsVisitedCities() async throws {
        mockNetwork.getVisitedLocationsCountryResult = .success(VisitedLocationsResponseDTO(locations: []))
        mockNetwork.getVisitedLocationsCityResult = .success(
            VisitedLocationsResponseDTO(locations: [
                VisitedLocationDTO(name: "Moscow", visitsCount: 5),
                VisitedLocationDTO(name: "Paris", visitsCount: 3),
                VisitedLocationDTO(name: "Berlin", visitsCount: 1)
            ])
        )

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.visitedCities.count, 3)
        XCTAssertEqual(sut.visitedCities.first?.name, "Moscow")
        XCTAssertTrue(sut.visitedCountries.isEmpty)
    }

    func test_dispatch_loadStats_callsVisitedLocationsWithCorrectTypes() async throws {
        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertTrue(mockNetwork.getVisitedLocationsCallTypes.contains("Country"))
        XCTAssertTrue(mockNetwork.getVisitedLocationsCallTypes.contains("City"))
    }

    // MARK: - isLoading

    func test_dispatch_loadStats_setsIsLoadingTrueWhileRunning() async throws {
        sut.dispatch(.loadStats)

        XCTAssertTrue(sut.isLoading)

        try await waitForNotLoading()

        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_loadStats_whileAlreadyLoading_isIgnored() async throws {
        sut.dispatch(.loadStats)
        XCTAssertTrue(sut.isLoading)

        sut.dispatch(.loadStats)

        try await waitForNotLoading()

        XCTAssertEqual(mockNetwork.getProfileStatsCallCount, 1)
    }

    // MARK: - loadStats — errors

    func test_dispatch_loadStats_statsFailure_keepsZeroCounts() async throws {
        mockNetwork.getProfileStatsResult = .failure(URLError(.badServerResponse))

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.tripsCount, 0)
        XCTAssertEqual(sut.pinsCount, 0)
        XCTAssertEqual(sut.mediaCount, 0)
        XCTAssertEqual(sut.likesCount, 0)
        XCTAssertEqual(sut.dislikesCount, 0)
        XCTAssertEqual(sut.battlesCount, 0)
    }

    func test_dispatch_loadStats_countriesFailure_setsEmptyArray() async throws {
        mockNetwork.getVisitedLocationsCountryResult = .failure(URLError(.badServerResponse))
        mockNetwork.getVisitedLocationsCityResult = .success(
            VisitedLocationsResponseDTO(locations: [VisitedLocationDTO(name: "Moscow")])
        )

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertTrue(sut.visitedCountries.isEmpty)
        XCTAssertEqual(sut.visitedCities.count, 1)
    }

    func test_dispatch_loadStats_citiesFailure_setsEmptyArray() async throws {
        mockNetwork.getVisitedLocationsCountryResult = .success(
            VisitedLocationsResponseDTO(locations: [VisitedLocationDTO(name: "Russia")])
        )
        mockNetwork.getVisitedLocationsCityResult = .failure(URLError(.badServerResponse))

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertEqual(sut.visitedCountries.count, 1)
        XCTAssertTrue(sut.visitedCities.isEmpty)
    }

    func test_dispatch_loadStats_allRequestsFail_setsIsLoadingFalse() async throws {
        mockNetwork.getProfileStatsResult = .failure(URLError(.badServerResponse))
        mockNetwork.getVisitedLocationsCountryResult = .failure(URLError(.badServerResponse))
        mockNetwork.getVisitedLocationsCityResult = .failure(URLError(.badServerResponse))

        sut.dispatch(.loadStats)
        try await waitForNotLoading()

        XCTAssertFalse(sut.isLoading)
    }

    // MARK: - Helpers

    private func waitForNotLoading() async throws {
        for _ in 0..<60 {
            if !sut.isLoading { return }
            try await Task.sleep(nanoseconds: 20_000_000)
        }
        XCTFail("isLoading did not become false in time")
    }
}
