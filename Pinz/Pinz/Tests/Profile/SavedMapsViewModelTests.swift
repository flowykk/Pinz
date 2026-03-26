import XCTest
@testable import PinzProfile
import PinzBase

final class SavedMapsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: SavedMapsViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = SavedMapsViewModel()
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
