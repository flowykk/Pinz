import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class TripsListViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!

    override func setUp() {
        super.setUp()
        SelectedTripStorage.shared.clearSelection()
        mockRouter = MockRouter()
    }

    override func tearDown() {
        SelectedTripStorage.shared.clearSelection()
        super.tearDown()
    }

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
}
