import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class TripPinsListViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: TripPinsListViewModel!
    private let trip = Trip.stub()

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = TripPinsListViewModel(trip: trip)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_init_storesTrip() {
        XCTAssertEqual(sut.trip.id, trip.id)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_pinCreation_callsRouter() {
        sut.dispatch(.navigate(.pinCreation))
        XCTAssertTrue(mockRouter.navigatedToPinCreation)
    }

    func test_navigate_pinInfo_callsRouter() {
        let pin = trip.pins.first!
        sut.dispatch(.navigate(.pinInfo(pin)))
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, pin.name)
    }
}
