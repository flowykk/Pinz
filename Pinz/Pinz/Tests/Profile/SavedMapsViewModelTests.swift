import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class SavedMapsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: SavedMapsViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = SavedMapsViewModel(networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockNetwork = nil
        mockRouter = nil
        sut = nil
        super.tearDown()
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_fetchFavouriteTrips_populatesTrips() async throws {
        let dto = TripDTO(
            id: "fav-1", name: "Fav", description: nil, category: nil, season: nil,
            coverUrl: nil, ownerUserId: "u1", privacyLevel: "public", status: "published",
            isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 0,
            startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1, updatedAtUnix: 1
        )
        mockNetwork.getFavouriteTripsResult = .success([dto])
        try await sut.asyncDispatch(.fetchFavouriteTrips)
        XCTAssertEqual(sut.trips.count, 1)
        XCTAssertEqual(sut.trips.first?.id, "fav-1")
    }

    func test_selectTrip_navigatesToSavedTripDetail() {
        let trip = TripDTO(
            id: "fav-1", name: "Fav", description: nil, category: nil, season: nil,
            coverUrl: nil, ownerUserId: "u1", privacyLevel: "public", status: "published",
            isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 0,
            startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1, updatedAtUnix: 1
        ).toTrip()
        sut.dispatch(.selectTrip(trip))
        XCTAssertEqual(mockRouter.navigatedSavedTrip?.id, "fav-1")
    }
}
