import XCTest
@testable import PinzProfile
import PinzBase

final class StatisticsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: StatisticsViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = StatisticsViewModel()
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
