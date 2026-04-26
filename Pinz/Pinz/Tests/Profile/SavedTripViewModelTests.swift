import XCTest
@testable import PinzProfile
import PinzDomain

@MainActor
final class SavedTripViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!
    private var mockRouter: MockRouter!
    private var trip: Trip!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
        mockRouter = MockRouter()
        trip = MockNetworkService.makeTripForTests()
    }

    override func tearDown() {
        trip = nil
        mockNetwork = nil
        mockRouter = nil
        super.tearDown()
    }

    func test_loadTrip_mapsPinsAndUpdatesTrip() async {
        let sut = SavedTripViewModel(trip: trip, networkService: mockNetwork)
        await sut.loadTrip()
        XCTAssertFalse(sut.isLoading)
        XCTAssertNil(sut.loadError)
        XCTAssertEqual(sut.pins.count, 2)
        XCTAssertEqual(sut.trip.id, "trip-001")
        XCTAssertTrue(sut.isSaved)
    }

    func test_toggleSaved_removeOnSuccess_flipsIsSaved() async {
        let sut = SavedTripViewModel(trip: trip, networkService: mockNetwork)
        mockNetwork.removeTripFromFavouritesError = nil
        await sut.toggleSaved()
        XCTAssertFalse(sut.isSaved)
        XCTAssertEqual(mockNetwork.removeTripFromFavouritesCall, "trip-001")
    }

    func test_toggleSaved_removeOnError_keepsIsSaved() async {
        let sut = SavedTripViewModel(trip: trip, networkService: mockNetwork)
        struct E: Error {}
        mockNetwork.removeTripFromFavouritesError = E()
        await sut.toggleSaved()
        XCTAssertTrue(sut.isSaved)
    }

    func test_toggleSaved_addOnSuccess_fromUnsaved() async {
        let sut = SavedTripViewModel(trip: trip, networkService: mockNetwork)
        await sut.toggleSaved()
        XCTAssertFalse(sut.isSaved)
        await sut.toggleSaved()
        XCTAssertTrue(sut.isSaved)
        XCTAssertEqual(mockNetwork.addTripToFavouritesCall, "trip-001")
    }
}

private extension MockNetworkService {
    static func makeTripForTests() -> Trip {
        let dto = TripDTO(
            id: "trip-001", name: "Test Trip", description: nil, category: nil, season: nil,
            coverUrl: nil, ownerUserId: "user-001", privacyLevel: "public", status: "published",
            isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 12,
            startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_000_000
        )
        return dto.toTrip()
    }
}
