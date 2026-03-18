import XCTest
@testable import PinzProfile
import PinzBase
import PinzUI

final class AppearanceViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: AppearanceViewModel!

    private let testKey = PinzMapStyle.mapStyleKey

    override func setUp() {
        super.setUp()
        UserDefaults.standard.removeObject(forKey: testKey)
        mockRouter = MockRouter()
        sut = AppearanceViewModel()
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        UserDefaults.standard.removeObject(forKey: testKey)
        sut = nil
        super.tearDown()
    }

    func test_changeMapStyle_updatesState() {
        sut.dispatch(.changeMapStyle(.hybrid))
        XCTAssertEqual(sut.state.mapStyle, .hybrid)
    }

    func test_saveMapStyle_persistsToUserDefaults() {
        sut.dispatch(.changeMapStyle(.scheme))
        let saved = UserDefaults.standard.string(forKey: testKey)
        XCTAssertEqual(saved, PinzMapStyle.scheme.rawValue)
    }

    func test_loadMapStyle_restoresFromUserDefaults() {
        UserDefaults.standard.set(PinzMapStyle.hybrid.rawValue, forKey: testKey)
        let freshVM = AppearanceViewModel()
        XCTAssertEqual(freshVM.state.mapStyle, .hybrid)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
