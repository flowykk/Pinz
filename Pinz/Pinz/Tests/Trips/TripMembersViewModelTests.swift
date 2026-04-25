import XCTest
@testable import PinzTrips
import PinzBase

@MainActor
final class TripMembersViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: TripMembersViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = TripMembersViewModel()
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
