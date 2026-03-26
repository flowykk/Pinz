import XCTest
import CoreLocation
@testable import PinzPins
import PinzBase
import PinzDomain

final class PinPlaceChangeViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: PinPlaceChangeViewModel!
    private var savedCoordinate: CLLocationCoordinate2D?
    private let originalCoord = CLLocationCoordinate2D(latitude: 55.7, longitude: 37.6)

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        let pin = Pin(
            name: "Test",
            category: .custom(),
            medias: [],
            isPrivate: false,
            tags: [],
            coordinates: originalCoord
        )
        sut = PinPlaceChangeViewModel(pin: pin) { [weak self] coord in
            self?.savedCoordinate = coord
        }
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialCoordinate() {
        XCTAssertEqual(sut.currentCoordinate.latitude, originalCoord.latitude)
        XCTAssertEqual(sut.currentCoordinate.longitude, originalCoord.longitude)
    }

    func test_hasChanges_falseInitially() {
        XCTAssertFalse(sut.hasChanges)
    }

    func test_hasChanges_trueAfterSignificantChange() {
        sut.currentCoordinate = CLLocationCoordinate2D(latitude: 10.0, longitude: 20.0)
        XCTAssertTrue(sut.hasChanges)
    }

    func test_save_callsOnSaveAndPops() {
        let newCoord = CLLocationCoordinate2D(latitude: 10.0, longitude: 20.0)
        sut.currentCoordinate = newCoord
        sut.dispatch(.save)
        XCTAssertEqual(savedCoordinate?.latitude, newCoord.latitude)
        XCTAssertEqual(savedCoordinate?.longitude, newCoord.longitude)
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_back_callsPop() {
        sut.dispatch(.back)
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_reset_restoresOriginalCoordinate() {
        sut.currentCoordinate = CLLocationCoordinate2D(latitude: 10.0, longitude: 20.0)
        sut.dispatch(.reset)
        XCTAssertEqual(sut.currentCoordinate.latitude, originalCoord.latitude, accuracy: 0.0001)
        XCTAssertEqual(sut.currentCoordinate.longitude, originalCoord.longitude, accuracy: 0.0001)
    }

    // .update(MapCameraUpdateContext) is not directly testable since
    // MapCameraUpdateContext has no public initializer (it's system-provided via onMapCameraChange).
}
