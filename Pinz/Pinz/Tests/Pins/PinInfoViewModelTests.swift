import XCTest
import CoreLocation
@testable import PinzPins
import PinzBase
import PinzDomain

final class PinInfoViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: PinInfoViewModel!
    private let pin = Pin.stubs().first!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = PinInfoViewModel(pin: pin)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .info)
        XCTAssertFalse(sut.isEditing)
    }

    func test_edit_changesStateToEditing() {
        sut.dispatch(.edit)
        XCTAssertEqual(sut.state, .editing)
        XCTAssertTrue(sut.isEditing)
    }

    func test_edit_remembersPreviousState() {
        sut.dispatch(.edit)
        sut.dispatch(.endEdit)
        XCTAssertEqual(sut.state, .info)
    }

    func test_editFromGallery_returnsToGallery() {
        sut.state = .gallery
        sut.dispatch(.edit)
        sut.dispatch(.endEdit)
        XCTAssertEqual(sut.state, .gallery)
    }

    func test_addTag_appendsTag() {
        let tag = MediaTag(tag: "TestTag")
        sut.dispatch(.addTag(tag))
        XCTAssertTrue(sut.pin.tags.contains(where: { $0.tag == "TestTag" }))
    }

    func test_deleteTag_removesTag() {
        let tag = MediaTag(tag: "RemoveMe")
        sut.dispatch(.addTag(tag))
        sut.dispatch(.deleteTag(tag))
        XCTAssertFalse(sut.pin.tags.contains(where: { $0.tag == "RemoveMe" }))
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_changePlace_callsRouter() {
        sut.dispatch(.navigate(.changePlace))
        XCTAssertNotNil(mockRouter.navigatedPinPlaceChange)
    }

    func test_navigate_changePlace_action_updatesCoordinate() {
        sut.dispatch(.navigate(.changePlace))
        let newCoord = CLLocationCoordinate2D(latitude: 10, longitude: 20)
        mockRouter.navigatedPinPlaceChange?.action.action(newCoord)
        guard let coordinate = sut.pin.coordinates else {
            return XCTFail("Expected pin coordinates to be set")
        }
        XCTAssertEqual(coordinate.latitude, 10)
        XCTAssertEqual(coordinate.longitude, 20)
    }

    func test_navigate_mediaInfo_callsRouter() {
        let media = pin.medias.first!
        sut.dispatch(.navigate(.mediaInfo(media)))
        XCTAssertEqual(mockRouter.navigatedMediaInfo?.id, media.id)
    }

    // MARK: - onDisappear

    func test_onDisappear_withUpdateAction_callsAction() {
        var receivedPin: Pin?
        let sut = PinInfoViewModel(pin: pin, updateAction: PinUpdateAction { receivedPin = $0 })
        sut.onDisappear()
        XCTAssertEqual(receivedPin?.name, pin.name)
    }

    func test_onDisappear_withoutUpdateAction_doesNotCrash() {
        sut.onDisappear()
    }

    // MARK: - State.id / State.content

    func test_stateId_returnsItself() {
        XCTAssertEqual(PinInfoViewModel.State.info.id, .info)
        XCTAssertEqual(PinInfoViewModel.State.gallery.id, .gallery)
        XCTAssertEqual(PinInfoViewModel.State.editing.id, .editing)
    }

    func test_stateContent_isText() {
        let infoContent = PinInfoViewModel.State.info.content
        let galleryContent = PinInfoViewModel.State.gallery.content
        let editingContent = PinInfoViewModel.State.editing.content

        if case .text(let text) = infoContent {
            XCTAssertFalse(text.isEmpty)
        } else {
            XCTFail("Expected .text for .info state")
        }

        if case .text(let text) = galleryContent {
            XCTAssertFalse(text.isEmpty)
        } else {
            XCTFail("Expected .text for .gallery state")
        }

        if case .text = editingContent {
        } else {
            XCTFail("Expected .text for .editing state")
        }
    }
}
