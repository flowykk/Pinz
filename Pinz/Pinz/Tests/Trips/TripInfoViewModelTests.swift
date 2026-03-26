import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class TripInfoViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: TripInfoViewModel!
    private let trip = Trip.stub()

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = TripInfoViewModel(trip: trip)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.trip.id, trip.id)
    }

    func test_changeState_togglesFromDefaultToEditing() {
        sut.dispatch(.changeState)
        XCTAssertEqual(sut.state, .editing)
    }

    func test_changeState_togglesFromEditingToDefault() {
        sut.dispatch(.changeState)
        sut.dispatch(.changeState)
        XCTAssertEqual(sut.state, .default)
    }

    func test_setImage_updatesImage() {
        let image = UIImage()
        sut.dispatch(.setImage(image))
        XCTAssertNotNil(sut.trip.image)
    }

    func test_setImage_nil_doesNotClearExistingImage() {
        sut.trip.image = UIImage()
        sut.dispatch(.setImage(nil))
        XCTAssertNotNil(sut.trip.image)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_pinsList_callsRouter() {
        sut.dispatch(.navigate(.pinsList))
        XCTAssertEqual(mockRouter.navigatedPinsList?.id, trip.id)
    }

    func test_navigate_selectPins_callsRouter() {
        sut.dispatch(.navigate(.selectPins))
        XCTAssertEqual(mockRouter.navigatedSelectablePinsList?.id, trip.id)
    }
}
