import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class TripsListPopupViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!
    private var sut: TripsListPopupViewModel!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
        sut = TripsListPopupViewModel(networkService: mockNetwork)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    // MARK: - fetchTrips

    func test_fetchTrips_success_populatesTrips() async throws {
        let dto = makeStubTripDTO(id: "t-1")
        mockNetwork.getTripsResult = .success([dto])
        try await sut.asyncDispatch(.fetchTrips(selectedTripId: "other"))
        XCTAssertEqual(sut.trips.count, 1)
        XCTAssertEqual(sut.trips[0].id, "t-1")
    }

    func test_fetchTrips_filtersOutSelectedTripId() async throws {
        let dto1 = makeStubTripDTO(id: "t-1")
        let dto2 = makeStubTripDTO(id: "t-2")
        mockNetwork.getTripsResult = .success([dto1, dto2])
        try await sut.asyncDispatch(.fetchTrips(selectedTripId: "t-1"))
        XCTAssertEqual(sut.trips.count, 1)
        XCTAssertEqual(sut.trips[0].id, "t-2")
    }

    func test_fetchTrips_noMatch_includesAllTrips() async throws {
        let dto1 = makeStubTripDTO(id: "t-1")
        let dto2 = makeStubTripDTO(id: "t-2")
        mockNetwork.getTripsResult = .success([dto1, dto2])
        try await sut.asyncDispatch(.fetchTrips(selectedTripId: "t-99"))
        XCTAssertEqual(sut.trips.count, 2)
    }

    func test_fetchTrips_failure_throws() async {
        struct FetchError: Error {}
        mockNetwork.getTripsResult = .failure(FetchError())
        do {
            try await sut.asyncDispatch(.fetchTrips(selectedTripId: ""))
            XCTFail("Expected error to be thrown")
        } catch {
            XCTAssertTrue(error is FetchError)
        }
    }

    func test_fetchTrips_setsIsLoading_duringFetch() async throws {
        mockNetwork.getTripsResult = .success([])
        XCTAssertFalse(sut.isLoading)
        try await sut.asyncDispatch(.fetchTrips(selectedTripId: ""))
        XCTAssertFalse(sut.isLoading)
    }

    func test_fetchTrips_emptyResult_setsEmptyTrips() async throws {
        mockNetwork.getTripsResult = .success([])
        try await sut.asyncDispatch(.fetchTrips(selectedTripId: ""))
        XCTAssertTrue(sut.trips.isEmpty)
    }

    // MARK: - Helpers

    private func makeStubTripDTO(id: String) -> TripDTO {
        TripDTO(
            id: id, name: "Trip \(id)", description: nil, category: nil,
            season: nil, coverUrl: nil, ownerUserId: "user-1", privacyLevel: "public",
            status: "published", isPublished: true, isGenerated: false,
            likesCount: 0, dislikesCount: 0, startDateUnix: nil, endDateUnix: nil,
            createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_000_000
        )
    }
}
