import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistElementViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockService: MockNetworkService!
    private var sut: WishlistElementViewModel!

    private let stubElement = DesiredPlace(id: "1", name: "Мачу Пикчу", description: "Древний город в горах Перу.")

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockService = MockNetworkService()
        sut = WishlistElementViewModel(element: stubElement, networkService: mockService)
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

    func test_endEdit_withWhitespaceName_showsToastAndStaysEditing() {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }
        sut.dispatch(.edit)
        sut.element.name = "   "
        sut.dispatch(.endEdit)
        XCTAssertEqual(sut.state, .editing)
        XCTAssertEqual(toasts, [PinzBaseStrings.WishlistElement.Toast.nameEmpty])
    }

    func test_endEdit_withInvalidNameCharacters_showsToastAndStaysEditing() {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }
        sut.dispatch(.edit)
        sut.element.name = "Place_with_underscore"
        sut.dispatch(.endEdit)
        XCTAssertEqual(sut.state, .editing)
        XCTAssertEqual(toasts, [PinzBaseStrings.WishlistElement.Toast.nameInvalidChars])
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_endEdit_callsUpdateDesiredPlace() async throws {
        let expectation = expectation(description: "updateDesiredPlace called")
        mockService.updateDesiredPlaceResult = .success(
            DesiredPlaceDTO(id: stubElement.id, name: "Updated", description: "Updated desc", imageUrl: nil, createdAt: 0)
        )

        sut.dispatch(.edit)
        sut.dispatch(.endEdit)

        Task {
            while mockService.updateDesiredPlaceCall == nil {
                await Task.yield()
            }
            expectation.fulfill()
        }

        await fulfillment(of: [expectation], timeout: 2.0)

        XCTAssertNotNil(mockService.updateDesiredPlaceCall)
        XCTAssertEqual(mockService.updateDesiredPlaceCall?.placeId, stubElement.id)
        XCTAssertEqual(mockService.updateDesiredPlaceCall?.name, stubElement.name)
        XCTAssertNil(mockService.updateDesiredPlaceCall?.imageS3Key)
    }

    func test_delete_callsDeleteAndPops() async throws {
        let expectation = expectation(description: "delete and pop called")

        sut.dispatch(.delete)

        Task {
            while mockService.deleteDesiredPlaceCall == nil {
                await Task.yield()
            }
            expectation.fulfill()
        }

        await fulfillment(of: [expectation], timeout: 2.0)

        XCTAssertEqual(mockService.deleteDesiredPlaceCall, stubElement.id)
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_endEdit_onUpdateFailure_showsToast() async {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }
        mockService.updateDesiredPlaceResult = .failure(URLError(.notConnectedToInternet))

        sut.dispatch(.edit)
        sut.dispatch(.endEdit)

        let exp = expectation(description: "toast")
        Task {
            while toasts.isEmpty { await Task.yield() }
            exp.fulfill()
        }
        await fulfillment(of: [exp], timeout: 2.0)

        XCTAssertEqual(toasts, [PinzBaseStrings.WishlistElement.Toast.updateFailed])
    }

    func test_delete_onFailure_showsToastAndDoesNotPop() async {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }
        mockService.deleteDesiredPlaceError = URLError(.notConnectedToInternet)

        sut.dispatch(.delete)

        let exp = expectation(description: "toast")
        Task {
            while toasts.isEmpty { await Task.yield() }
            exp.fulfill()
        }
        await fulfillment(of: [exp], timeout: 2.0)

        XCTAssertEqual(toasts, [PinzBaseStrings.WishlistElement.Toast.deleteFailed])
        XCTAssertEqual(mockRouter.popCallCount, 0)
    }
}
