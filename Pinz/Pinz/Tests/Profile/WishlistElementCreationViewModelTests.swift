import XCTest
import UIKit
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistElementCreationViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockService: MockNetworkService!
    private var sut: WishlistElementCreationViewModel!
    private var createdElement: DesiredPlace?

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockService = MockNetworkService()
        sut = WishlistElementCreationViewModel(onCreated: { [weak self] element in
            self?.createdElement = element
        }, networkService: mockService)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .name)
        XCTAssertEqual(sut.name, "")
        XCTAssertEqual(sut.description, "")
        XCTAssertNil(sut.image)
    }

    func test_isCompleteButtonDisabled_whenNameEmpty() {
        sut.name = ""
        XCTAssertTrue(sut.isCompleteButtonDisabled)
    }

    func test_isCompleteButtonDisabled_whenNameFilled() {
        sut.name = "Paris"
        XCTAssertFalse(sut.isCompleteButtonDisabled)
    }

    func test_continue_fromName_goesToDescription() {
        sut.name = "Paris"
        sut.dispatch(.continue)
        XCTAssertEqual(sut.state, .description)
    }

    func test_isCompleteButtonDisabled_inDescriptionState_whenEmpty() {
        sut.name = "Paris"
        sut.dispatch(.continue)
        XCTAssertTrue(sut.isCompleteButtonDisabled)
    }

    func test_continue_fromDescription_goesToPhoto() {
        sut.name = "Paris"
        sut.dispatch(.continue)
        sut.description = "Beautiful city"
        sut.dispatch(.continue)
        XCTAssertEqual(sut.state, .photo)
    }

    func test_isCompleteButtonDisabled_inPhotoState_whenNoImage() {
        sut.name = "Paris"
        sut.dispatch(.continue)
        sut.description = "Beautiful city"
        sut.dispatch(.continue)
        XCTAssertTrue(sut.isCompleteButtonDisabled)
    }

    func test_continue_fromPhoto_withImage_callsOnCreatedAndPops() async throws {
        let expectation = expectation(description: "onCreated called")
        sut = WishlistElementCreationViewModel(onCreated: { [weak self] element in
            self?.createdElement = element
            expectation.fulfill()
        }, networkService: mockService)
        sut.setRouter(mockRouter)

        sut.name = "Paris"
        sut.dispatch(.continue)
        sut.description = "Beautiful city"
        sut.dispatch(.continue)

        let renderer = UIGraphicsImageRenderer(size: CGSize(width: 1, height: 1))
        sut.image = renderer.image { ctx in
            UIColor.red.setFill()
            ctx.fill(CGRect(x: 0, y: 0, width: 1, height: 1))
        }
        sut.dispatch(.continue)

        await fulfillment(of: [expectation], timeout: 2.0)

        XCTAssertEqual(createdElement?.name, MockNetworkService.stubDesiredPlace.name)
        XCTAssertEqual(mockRouter.popCallCount, 1)
        XCTAssertNotNil(mockService.createDesiredPlaceCall)
        XCTAssertEqual(mockService.createDesiredPlaceCall?.name, "Paris")
        XCTAssertNotNil(mockService.requestDesiredPlaceImageUploadCall)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
