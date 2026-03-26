import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class PostPreviewViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: PostPreviewViewModel!
    private let trip = Trip.stub()

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = PostPreviewViewModel(trip: trip, selectedPins: trip.pins)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_init_storesTrip() {
        XCTAssertEqual(sut.trip.id, trip.id)
    }

    func test_init_selectedPinsStripsPrivateMedias() {
        let pinsWithMedia = trip.pins
        let sut = PostPreviewViewModel(trip: trip, selectedPins: pinsWithMedia)
        for pin in sut.selectedPins {
            XCTAssertTrue(pin.medias.allSatisfy { !$0.isPrivate })
        }
    }

    func test_init_preservesPinCount() {
        XCTAssertEqual(sut.selectedPins.count, trip.pins.count)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back()))
        XCTAssertEqual(mockRouter.popCallCount, 1)
        XCTAssertEqual(mockRouter.lastPopByCount, 1)
    }

    func test_navigate_back_withCustomDepth_callsPopByCount() {
        sut.dispatch(.navigate(.back(by: 3)))
        XCTAssertEqual(mockRouter.lastPopByCount, 3)
    }

    func test_navigate_pinInfo_callsRouter() {
        let pin = trip.pins.first!
        sut.dispatch(.navigate(.pinInfo(pin)))
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, pin.name)
    }
}
