import XCTest
@testable import PinzProfile
import PinzBase

final class NotificationsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: NotificationsViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = NotificationsViewModel()
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
