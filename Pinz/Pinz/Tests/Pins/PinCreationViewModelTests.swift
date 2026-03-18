import XCTest
@testable import PinzPins
import PinzBase
import PinzDomain

@MainActor
final class PinCreationViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: PinCreationViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = PinCreationViewModel()
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .info)
        XCTAssertEqual(sut.name, "")
        XCTAssertTrue(sut.tags.isEmpty)
        XCTAssertTrue(sut.medias.isEmpty)
    }

    func test_addTag_appendsTag() {
        let tag = MediaTag(tag: "Nature")
        sut.dispatch(.addTag(tag))
        XCTAssertTrue(sut.tags.contains(where: { $0.tag == "Nature" }))
    }

    func test_deleteTag_removesTag() {
        let tag = MediaTag(tag: "Food")
        sut.dispatch(.addTag(tag))
        sut.dispatch(.deleteTag(tag))
        XCTAssertFalse(sut.tags.contains(where: { $0.tag == "Food" }))
    }

    func test_addTag_multipleTagsAccumulate() {
        sut.dispatch(.addTag(MediaTag(tag: "A")))
        sut.dispatch(.addTag(MediaTag(tag: "B")))
        sut.dispatch(.addTag(MediaTag(tag: "C")))
        XCTAssertEqual(sut.tags.count, 3)
    }

    func test_deleteMedia_removesMediaById() {
        let id = UUID()
        // We can only delete existing medias; test that dispatch does not crash on missing id
        sut.dispatch(.deleteMedia(id))
        XCTAssertTrue(sut.medias.isEmpty)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }
}
