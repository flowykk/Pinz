import XCTest
@testable import PinzFeed
import PinzBase

final class FeedViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: FeedViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = FeedViewModel()
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
