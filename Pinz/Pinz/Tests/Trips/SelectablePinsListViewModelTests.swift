import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class SelectablePinsListViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: SelectablePinsListViewModel!
    private var trip: Trip!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        trip = Trip.stub()
        sut = SelectablePinsListViewModel(trip: trip)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_init_storesTrip() {
        XCTAssertEqual(sut.trip.id, trip.id)
    }

    func test_init_noSelectedPins() {
        XCTAssertTrue(sut.selectedPins.isEmpty)
    }

    func test_init_publicPinsSortedFirst() {
        let publicPins = sut.pins.filter { !$0.isPrivate }
        let privatePins = sut.pins.filter { $0.isPrivate }
        let firstPrivateIndex = sut.pins.firstIndex(where: { $0.isPrivate }) ?? sut.pins.count
        let lastPublicIndex = sut.pins.lastIndex(where: { !$0.isPrivate }) ?? -1
        if !publicPins.isEmpty && !privatePins.isEmpty {
            XCTAssertLessThan(lastPublicIndex, firstPrivateIndex)
        }
    }

    func test_select_addsPin() {
        let pin = sut.pins.first(where: { !$0.isPrivate })!
        sut.dispatch(.select(pin))
        XCTAssertTrue(sut.selectedPins.contains(pin.id))
    }

    func test_select_twice_removesPin() {
        let pin = sut.pins.first(where: { !$0.isPrivate })!
        sut.dispatch(.select(pin))
        sut.dispatch(.select(pin))
        XCTAssertFalse(sut.selectedPins.contains(pin.id))
    }

    func test_isSelected_returnsTrue_whenPinSelected() {
        let pin = sut.pins.first(where: { !$0.isPrivate })!
        sut.dispatch(.select(pin))
        XCTAssertTrue(sut.isSelected(pin))
    }

    func test_isSelected_returnsFalse_whenPinNotSelected() {
        let pin = sut.pins.first(where: { !$0.isPrivate })!
        XCTAssertFalse(sut.isSelected(pin))
    }

    func test_selectAll_selectsAllPublicPins() {
        let publicPins = sut.pins.filter { !$0.isPrivate }
        sut.dispatch(.selectAll)
        XCTAssertEqual(sut.selectedPins.count, publicPins.count)
    }

    func test_selectAll_whenAllSelected_deselectsAll() {
        sut.dispatch(.selectAll)
        sut.dispatch(.selectAll)
        XCTAssertTrue(sut.selectedPins.isEmpty)
    }

    func test_allSelected_falseWhenEmpty() {
        XCTAssertFalse(sut.allSelected)
    }

    func test_allSelected_trueWhenAllPublicPinsSelected() {
        let publicPins = sut.pins.filter { !$0.isPrivate }
        guard !publicPins.isEmpty else { return }
        sut.dispatch(.selectAll)
        XCTAssertTrue(sut.allSelected)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_pinInfo_callsRouter() {
        let pin = trip.pins.first!
        sut.dispatch(.navigate(.pinInfo(pin)))
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, pin.name)
    }

    func test_navigate_postPreview_callsRouterWithSelectedPins() {
        let publicPins = sut.pins.filter { !$0.isPrivate }
        guard !publicPins.isEmpty else { return }
        sut.dispatch(.select(publicPins.first!))
        sut.dispatch(.navigate(.postPreview))
        XCTAssertNotNil(mockRouter.navigatedPostPreview)
        XCTAssertEqual(mockRouter.navigatedPostPreview?.pins.count, 1)
    }
}
