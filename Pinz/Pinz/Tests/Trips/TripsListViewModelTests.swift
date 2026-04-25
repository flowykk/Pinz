import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

@MainActor
final class TripsListViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!

    override func setUp() {
        super.setUp()
        SelectedTripStorage.shared.clearSelection()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
    }

    override func tearDown() {
        SelectedTripStorage.shared.clearSelection()
        mockNetwork = nil
        super.tearDown()
    }

    // MARK: - Init

    func test_init_filtersOutSelectedTrip() {
        let trips = Trip.stubs()
        let selected = trips[0]
        SelectedTripStorage.shared.selectTrip(id: selected.id)
        let sut = TripsListViewModel(trips: trips)
        XCTAssertFalse(sut.trips.contains(where: { $0.id == selected.id }))
    }

    func test_init_withNoSelection_includesAllTrips() {
        let trips = Trip.stubs()
        let sut = TripsListViewModel(trips: trips)
        XCTAssertEqual(sut.trips.count, trips.count)
    }

    // MARK: - dispatch

    func test_navigate_back_callsPop() {
        let sut = TripsListViewModel(trips: [])
        sut.setRouter(mockRouter)
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_selectTrip_savesToStorageAndPopsBy2() {
        let trips = Trip.stubs()
        let sut = TripsListViewModel(trips: trips)
        sut.setRouter(mockRouter)
        sut.dispatch(.selectTrip(trips[0]))
        XCTAssertEqual(SelectedTripStorage.shared.selectedTripID, trips[0].id)
        XCTAssertEqual(mockRouter.lastPopByCount, 2)
    }

    // MARK: - asyncDispatch fetchTrips

    func test_asyncDispatch_fetchTrips_success_setsTrips() async throws {
        let dto = stubTripDTO(id: "fetched-1", name: "Fetched Trip")
        mockNetwork.getTripsResult = .success([dto])
        let sut = TripsListViewModel(trips: [], networkService: mockNetwork)

        try await sut.asyncDispatch(.fetchTrips)

        XCTAssertEqual(sut.trips.count, 1)
        XCTAssertEqual(sut.trips[0].id, "fetched-1")
    }

    func test_asyncDispatch_fetchTrips_success_filtersSelectedTrip() async throws {
        let dto1 = stubTripDTO(id: "trip-a")
        let dto2 = stubTripDTO(id: "trip-b")
        mockNetwork.getTripsResult = .success([dto1, dto2])
        SelectedTripStorage.shared.selectTrip(id: "trip-a")
        let sut = TripsListViewModel(trips: [], networkService: mockNetwork)

        try await sut.asyncDispatch(.fetchTrips)

        XCTAssertEqual(sut.trips.count, 1)
        XCTAssertEqual(sut.trips[0].id, "trip-b")
    }

    func test_asyncDispatch_fetchTrips_success_setsIsLoadingFalse() async throws {
        mockNetwork.getTripsResult = .success([])
        let sut = TripsListViewModel(trips: [], networkService: mockNetwork)

        try await sut.asyncDispatch(.fetchTrips)

        XCTAssertFalse(sut.isLoading)
    }

    func test_asyncDispatch_fetchTrips_failure_throws() async {
        mockNetwork.getTripsResult = .failure(URLError(.badServerResponse))
        let sut = TripsListViewModel(trips: [], networkService: mockNetwork)

        do {
            try await sut.asyncDispatch(.fetchTrips)
            XCTFail("Expected error to be thrown")
        } catch {
            XCTAssertTrue(error is URLError)
        }
    }

    func test_asyncDispatch_fetchTrips_failure_leavesIsLoadingTrue() async {
        mockNetwork.getTripsResult = .failure(URLError(.badServerResponse))
        let sut = TripsListViewModel(trips: [], networkService: mockNetwork)

        try? await sut.asyncDispatch(.fetchTrips)

        XCTAssertTrue(sut.isLoading)
    }

    func test_asyncDispatch_fetchTrips_emptyResult_setsEmptyTrips() async throws {
        let sut = TripsListViewModel(trips: Trip.stubs(), networkService: mockNetwork)
        mockNetwork.getTripsResult = .success([])

        try await sut.asyncDispatch(.fetchTrips)

        XCTAssertTrue(sut.trips.isEmpty)
    }

    // MARK: - Helpers

    private func stubTripDTO(id: String, name: String = "Trip") -> TripDTO {
        TripDTO(
            id: id, name: name, description: nil, category: nil, season: nil,
            coverUrl: nil, ownerUserId: "user-1", privacyLevel: "public", status: "published",
            isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 0,
            startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_000_000
        )
    }
}
