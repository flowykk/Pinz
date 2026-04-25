import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class PreprocessedRawPinsViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: PreprocessedRawPinsViewModel!

    private let tripId = "trip-raw-001"

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        let pins = RawPins(pins: [
            RawPin(id: "pin-1", medias: [
                RawPinMedia(id: "media-1", url: "https://x.com/1.jpg", type: .image),
                RawPinMedia(id: "media-2", url: "https://x.com/2.jpg", type: .image)
            ]),
            RawPin(id: "pin-2", medias: [
                RawPinMedia(id: "media-3", url: "https://x.com/3.jpg", type: .image)
            ])
        ])
        sut = PreprocessedRawPinsViewModel(tripId: tripId, pins: pins, networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockRouter = nil
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    // MARK: - deleteMedia

    func test_dispatch_deleteMedia_removesMediaFromPin() {
        let media = sut.pins.pins[0].medias[0]
        sut.dispatch(.deleteMedia(media, fromPin: "pin-1"))
        XCTAssertEqual(sut.pins.pins[0].medias.count, 1)
        XCTAssertFalse(sut.pins.pins[0].medias.contains { $0.id == "media-1" })
    }

    func test_dispatch_deleteMedia_unknownPin_doesNothing() {
        let media = sut.pins.pins[0].medias[0]
        sut.dispatch(.deleteMedia(media, fromPin: "nonexistent"))
        XCTAssertEqual(sut.pins.pins[0].medias.count, 2)
    }

    // MARK: - mergePins

    func test_dispatch_mergePins_combinesMediaFromSecondIntoFirst() {
        sut.dispatch(.mergePins(firstIndex: 0, secondIndex: 1))
        XCTAssertEqual(sut.pins.pins.count, 1)
        XCTAssertEqual(sut.pins.pins[0].medias.count, 3)
    }

    func test_dispatch_mergePins_sameIndex_doesNothing() {
        let initialCount = sut.pins.pins.count
        sut.dispatch(.mergePins(firstIndex: 0, secondIndex: 0))
        XCTAssertEqual(sut.pins.pins.count, initialCount)
    }

    func test_dispatch_mergePins_outOfBounds_doesNothing() {
        let initialCount = sut.pins.pins.count
        sut.dispatch(.mergePins(firstIndex: 0, secondIndex: 99))
        XCTAssertEqual(sut.pins.pins.count, initialCount)
    }

    // MARK: - addPin

    func test_dispatch_addPin_appendsEmptyPin() {
        let initialCount = sut.pins.pins.count
        sut.dispatch(.addPin)
        XCTAssertEqual(sut.pins.pins.count, initialCount + 1)
        XCTAssertTrue(sut.pins.pins.last!.medias.isEmpty)
    }

    // MARK: - moveMedia

    func test_dispatch_moveMedia_movesMediaBetweenPins() {
        let media = sut.pins.pins[0].medias[0]
        sut.dispatch(.moveMedia(media, fromPin: 0, toPin: 1))
        XCTAssertFalse(sut.pins.pins[0].medias.contains { $0.id == media.id })
        XCTAssertTrue(sut.pins.pins[1].medias.contains { $0.id == media.id })
    }

    func test_dispatch_moveMedia_samePin_doesNothing() {
        let initialMedia0 = sut.pins.pins[0].medias
        let media = sut.pins.pins[0].medias[0]
        sut.dispatch(.moveMedia(media, fromPin: 0, toPin: 0))
        XCTAssertEqual(sut.pins.pins[0].medias.map(\.id), initialMedia0.map(\.id))
    }

    func test_dispatch_moveMedia_outOfBounds_doesNothing() {
        let initialCount = sut.pins.pins[0].medias.count
        let media = sut.pins.pins[0].medias[0]
        sut.dispatch(.moveMedia(media, fromPin: 0, toPin: 99))
        XCTAssertEqual(sut.pins.pins[0].medias.count, initialCount)
    }

    // MARK: - Navigation

    func test_dispatch_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_dispatch_navigate_review_callsRouter() {
        let domainPins = Pin.stubs()
        sut.dispatch(.navigate(.review(tripId: tripId, pins: domainPins)))
        XCTAssertEqual(mockRouter.navigatedTripCreationReview?.tripId, tripId)
        XCTAssertEqual(mockRouter.navigatedTripCreationReview?.pins.count, domainPins.count)
    }

    // MARK: - asyncDispatch continue

    func test_asyncDispatch_continue_success_navigatesToReview() async throws {
        mockNetwork.applyGroupsAndProcessResult = .success(
            ApplyGroupsAndProcessDTO(message: "ok", status: "processing")
        )
        mockNetwork.waitForTripProcessingCompletedResult = .success(())
        mockNetwork.getTripReviewResult = .success(
            GetTripReviewDTO(tripId: tripId, status: "DRAFT_FINAL_REVIEW", pins: [], similar: [])
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockRouter.navigatedTripCreationReview?.tripId, tripId)
        XCTAssertFalse(sut.isLoading)
    }

    func test_asyncDispatch_continue_applyGroupsFails_throws() async {
        struct ApplyError: Error {}
        mockNetwork.applyGroupsAndProcessResult = .failure(ApplyError())

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is ApplyError)
        }
        XCTAssertFalse(sut.isLoading)
    }
}
