import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class TripViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: TripViewModel!

    override func setUp() {
        super.setUp()
        SelectedTripStorage.shared.clearSelection()
        mockRouter = MockRouter()
        sut = TripViewModel(trip: nil)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        SelectedTripStorage.shared.clearSelection()
        sut = nil
        super.tearDown()
    }

    // MARK: - Initial state

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.routePinIndex, 0)
        XCTAssertNil(sut.trip)
        XCTAssertNil(sut.selectedPin)
    }

    // MARK: - selectTrip

    func test_selectTrip_updatesTrip() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        XCTAssertEqual(sut.trip?.id, trip.id)
    }

    func test_selectTrip_resetsState() {
        sut.dispatch(.toggleRouteState)
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.routePinIndex, 0)
    }

    func test_selectTrip_clearsSelectedPin() {
        let pins = Pin.stubs()
        sut.dispatch(.selectPin(pin: pins.first))
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        XCTAssertNil(sut.selectedPin)
    }

    func test_selectTrip_persitsToStorage() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        XCTAssertEqual(SelectedTripStorage.shared.selectedTripID, trip.id)
    }

    // MARK: - clearSelection

    func test_clearSelection_resetsState() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.toggleRouteState)
        sut.dispatch(.selectPin(pin: trip.pins.first))
        sut.dispatch(.clearSelection)

        XCTAssertNil(sut.trip)
        XCTAssertNil(sut.selectedPin)
        XCTAssertFalse(sut.isLoading)
        XCTAssertEqual(sut.routePinIndex, 0)
        XCTAssertEqual(sut.state, .default)
        XCTAssertNil(SelectedTripStorage.shared.selectedTripID)
    }

    // MARK: - selectPin / unselectPin

    func test_selectPin_setsSelectedPin() {
        let pin = Pin.stubs().first!
        sut.dispatch(.selectPin(pin: pin))
        XCTAssertEqual(sut.selectedPin?.name, pin.name)
    }

    func test_selectPin_nil_clearsSelectedPin() {
        let pin = Pin.stubs().first!
        sut.dispatch(.selectPin(pin: pin))
        sut.dispatch(.selectPin(pin: nil))
        XCTAssertNil(sut.selectedPin)
    }

    func test_unselectPin_withSelectedPin_navigatesToPinInfo() {
        let pin = Pin.stubs().first!
        sut.dispatch(.selectPin(pin: pin))
        sut.dispatch(.unselectPin)
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, pin.name)
        XCTAssertNil(sut.selectedPin)
    }

    func test_unselectPin_withNoSelectedPin_doesNotNavigate() {
        sut.dispatch(.unselectPin)
        XCTAssertNil(mockRouter.navigatedPinInfo)
    }

    // MARK: - toggleRouteState

    func test_toggleRouteState_changesStateToRoute() {
        sut.dispatch(.toggleRouteState)
        XCTAssertEqual(sut.state, .route)
    }

    func test_toggleRouteState_twice_returnsToDefault() {
        sut.dispatch(.toggleRouteState)
        sut.dispatch(.toggleRouteState)
        XCTAssertEqual(sut.state, .default)
    }

    // MARK: - nextPin / previousPin

    func test_nextPin_incrementsIndex() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.toggleRouteState)
        sut.dispatch(.nextPin)
        XCTAssertEqual(sut.routePinIndex, 1)
    }

    func test_nextPin_doesNotExceedLastPin() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.toggleRouteState)
        let lastIndex = trip.pins.count - 1
        for _ in 0..<lastIndex + 5 {
            sut.dispatch(.nextPin)
        }
        XCTAssertEqual(sut.routePinIndex, lastIndex)
    }

    func test_previousPin_doesNotGoBelowZero() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.toggleRouteState)
        sut.dispatch(.previousPin)
        XCTAssertEqual(sut.routePinIndex, 0)
    }

    func test_nextThenPrevious_restoresIndex() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.toggleRouteState)
        sut.dispatch(.nextPin)
        sut.dispatch(.previousPin)
        XCTAssertEqual(sut.routePinIndex, 0)
    }

    // MARK: - sortedPins

    func test_sortedPins_sortsByStartDateAscending() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        let sorted = sut.sortedPins
        let dates = sorted.compactMap { $0.startDate }
        XCTAssertEqual(dates, dates.sorted())
    }

    // MARK: - Navigate

    func test_navigate_tripInfo_callsRouter() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.navigate(.tripInfo))
        XCTAssertEqual(mockRouter.navigatedTripInfo?.id, trip.id)
    }

    func test_navigate_tripInfo_setsUpdateHandler() {
        let trip = Trip.stub()
        sut.dispatch(.selectTrip(trip))
        sut.dispatch(.navigate(.tripInfo))
        XCTAssertNotNil(mockRouter.tripInfoUpdateHandler)
    }

    func test_navigate_feed_callsRouter() {
        sut.dispatch(.navigate(.feed))
        XCTAssertTrue(mockRouter.navigatedToFeed)
    }

    func test_navigate_members_callsRouter() {
        sut.dispatch(.navigate(.members))
        XCTAssertTrue(mockRouter.navigatedToTripMembers)
    }

    func test_navigate_pinCreation_callsRouter() {
        sut.dispatch(.navigate(.pinCreation))
        XCTAssertTrue(mockRouter.navigatedToPinCreation)
    }
}
