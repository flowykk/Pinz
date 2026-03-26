import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistElementViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: WishlistElementViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = WishlistElementViewModel(element: WishlistElement.stubs[0])
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
    }

    func test_edit_changesStateToEditing() {
        sut.dispatch(.edit)
        XCTAssertEqual(sut.state, .editing)
    }

    func test_endEdit_changesStateToDefault() {
        sut.dispatch(.edit)
        sut.dispatch(.endEdit)
        XCTAssertEqual(sut.state, .default)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
