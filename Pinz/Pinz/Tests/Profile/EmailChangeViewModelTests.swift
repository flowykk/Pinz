import XCTest
@testable import PinzProfile
import PinzBase

final class EmailChangeViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: EmailChangeViewModel!
    private var successCallbackEmail: String?

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = EmailChangeViewModel(email: "old@example.com") { [weak self] email in
            self?.successCallbackEmail = email
        }
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .firstCode)
        XCTAssertEqual(sut.email, "old@example.com")
    }

    func test_continue_fromFirstCode_goesToEmail() {
        sut.dispatch(.continue)
        XCTAssertEqual(sut.state, .email)
    }

    func test_continue_fromFirstCode_resetsCode() {
        sut.code = ["1", "2", "3", "4"]
        sut.dispatch(.continue)
        XCTAssertEqual(sut.code, ["", "", "", ""])
    }

    func test_continue_fromEmail_goesToSecondCode() {
        sut.dispatch(.continue)
        sut.dispatch(.continue)
        XCTAssertEqual(sut.state, .secondCode)
    }

    func test_continue_fromSecondCode_callsSuccessAction() {
        sut.email = "new@example.com"
        sut.dispatch(.continue)
        sut.dispatch(.continue)
        sut.dispatch(.continue)
        XCTAssertEqual(successCallbackEmail, "new@example.com")
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
