import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistElementCreationViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: WishlistElementCreationViewModel!
    private var createdElement: WishlistElement?

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = WishlistElementCreationViewModel { [weak self] element in
            self?.createdElement = element
        }
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

    func test_continue_fromPhoto_withImage_callsOnCreatedAndPops() {
        sut.name = "Paris"
        sut.dispatch(.continue)
        sut.description = "Beautiful city"
        sut.dispatch(.continue)
        sut.image = UIImage()
        sut.dispatch(.continue)

        XCTAssertNotNil(createdElement)
        XCTAssertEqual(createdElement?.title, "Paris")
        XCTAssertEqual(createdElement?.description, "Beautiful city")
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
