import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain

final class PostPreviewViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: PostPreviewViewModel!
    private let trip = Trip.stub()

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = PostPreviewViewModel(trip: trip, selectedPins: trip.pins, networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        mockNetwork = nil
        super.tearDown()
    }

    // MARK: - Init

    func test_init_storesTrip() {
        XCTAssertEqual(sut.trip.id, trip.id)
    }

    func test_init_selectedPinsStripsPrivateMedias() {
        let pinsWithMedia = trip.pins
        let sut = PostPreviewViewModel(trip: trip, selectedPins: pinsWithMedia, networkService: mockNetwork)
        for pin in sut.selectedPins {
            XCTAssertTrue(pin.medias.allSatisfy { !$0.isPrivate })
        }
    }

    func test_init_preservesPinCount() {
        XCTAssertEqual(sut.selectedPins.count, trip.pins.count)
    }

    // MARK: - dispatch

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back()))
        XCTAssertEqual(mockRouter.popCallCount, 1)
        XCTAssertEqual(mockRouter.lastPopByCount, 1)
    }

    func test_navigate_back_withCustomDepth_callsPopByCount() {
        sut.dispatch(.navigate(.back(by: 3)))
        XCTAssertEqual(mockRouter.lastPopByCount, 3)
    }

    func test_navigate_pinInfo_callsRouter() {
        let pin = trip.pins.first!
        sut.dispatch(.navigate(.pinInfo(pin)))
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, pin.name)
    }

    // MARK: - asyncDispatch publish — success

    func test_asyncDispatch_publish_success_popsBy2() async {
        await sut.asyncDispatch(.publish)
        XCTAssertEqual(mockRouter.lastPopByCount, 2)
    }

    func test_asyncDispatch_publish_success_updatesTrip() async {
        let publishedDTO = TripDTO(
            id: trip.id, name: trip.name, description: nil, category: nil, season: nil,
            coverUrl: nil, ownerUserId: "u1", privacyLevel: "public", status: "published",
            isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 0,
            startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_001_000
        )
        mockNetwork.publishTripResult = .success(publishedDTO)

        await sut.asyncDispatch(.publish)

        XCTAssertTrue(sut.trip.isPublished)
        XCTAssertEqual(sut.trip.status, "published")
    }

    func test_asyncDispatch_publish_success_setsIsPublishingFalse() async {
        await sut.asyncDispatch(.publish)
        XCTAssertFalse(sut.isPublishing)
    }

    func test_asyncDispatch_publish_success_clearsPublishError() async {
        sut.publishError = "previous error"
        await sut.asyncDispatch(.publish)
        XCTAssertNil(sut.publishError)
    }

    // MARK: - asyncDispatch publish — failure

    func test_asyncDispatch_publish_failure_setsPublishError() async {
        mockNetwork.publishTripResult = .failure(URLError(.badServerResponse))
        await sut.asyncDispatch(.publish)
        XCTAssertNotNil(sut.publishError)
    }

    func test_asyncDispatch_publish_failure_callsOnError() async {
        mockNetwork.publishTripResult = .failure(URLError(.notConnectedToInternet))
        var receivedError: Error?

        await sut.asyncDispatch(.publish, onError: { receivedError = $0 })

        XCTAssertNotNil(receivedError)
        XCTAssertTrue(receivedError is URLError)
    }

    func test_asyncDispatch_publish_failure_setsIsPublishingFalse() async {
        mockNetwork.publishTripResult = .failure(URLError(.badServerResponse))
        await sut.asyncDispatch(.publish)
        XCTAssertFalse(sut.isPublishing)
    }

    func test_asyncDispatch_publish_failure_doesNotPop() async {
        mockNetwork.publishTripResult = .failure(URLError(.badServerResponse))
        await sut.asyncDispatch(.publish)
        XCTAssertEqual(mockRouter.popCallCount, 0)
    }

    func test_asyncDispatch_publish_withNilOnError_doesNotCrash() async {
        mockNetwork.publishTripResult = .failure(URLError(.badServerResponse))
        await sut.asyncDispatch(.publish, onError: nil)
        XCTAssertNotNil(sut.publishError)
    }

    // MARK: - publishTrip — normalizedPinIds

    func test_publishTrip_usesPinNameAsId() async {
        let pin = makePin(name: "my-pin-name")
        sut = PostPreviewViewModel(trip: trip, selectedPins: [pin], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        await sut.asyncDispatch(.publish)

        XCTAssertEqual(mockNetwork.publishTripCall?.pinIds, ["my-pin-name"])
    }

    func test_publishTrip_filtersEmptyPinNames() async {
        let pinWithEmptyName = makePin(name: "", id: "")
        let pinWithValidName = makePin(name: "valid-pin")
        sut = PostPreviewViewModel(trip: trip, selectedPins: [pinWithEmptyName, pinWithValidName], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        await sut.asyncDispatch(.publish)

        XCTAssertEqual(mockNetwork.publishTripCall?.pinIds, ["valid-pin"])
    }

    func test_publishTrip_deduplicatesPinsWithSameName() async {
        let pin1 = makePin(name: "same-name")
        let pin2 = makePin(name: "same-name")
        sut = PostPreviewViewModel(trip: trip, selectedPins: [pin1, pin2], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        await sut.asyncDispatch(.publish)

        XCTAssertEqual(mockNetwork.publishTripCall?.pinIds, ["same-name"])
    }

    func test_publishTrip_preservesOrderOfUniqueNames() async {
        let pin1 = makePin(name: "first")
        let pin2 = makePin(name: "second")
        let pin3 = makePin(name: "first")
        sut = PostPreviewViewModel(trip: trip, selectedPins: [pin1, pin2, pin3], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        await sut.asyncDispatch(.publish)

        XCTAssertEqual(mockNetwork.publishTripCall?.pinIds, ["first", "second"])
    }

    // MARK: - publishWhole flag

    func test_publishTrip_publishWhole_whenAllPinsSelected() async {
        // sut is created with selectedPins = trip.pins in setUp
        await sut.asyncDispatch(.publish)
        XCTAssertEqual(mockNetwork.publishTripCall?.publishWhole, true)
    }

    func test_publishTrip_publishPartial_whenSubsetSelected() async {
        guard trip.pins.count > 1 else { return }
        sut = PostPreviewViewModel(trip: trip, selectedPins: [trip.pins.first!], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        await sut.asyncDispatch(.publish)

        XCTAssertEqual(mockNetwork.publishTripCall?.publishWhole, false)
    }

    func test_publishTrip_sendsCorrectTripId() async {
        await sut.asyncDispatch(.publish)
        XCTAssertEqual(mockNetwork.publishTripCall?.id, trip.id)
    }

    // MARK: - Helpers

    private func makePin(name: String = "Test Pin", id: String? = nil, serverId: String? = nil) -> Pin {
        let resolvedServerId = serverId ?? (name.isEmpty ? nil : name)
        return Pin(
            id: id,
            name: name,
            category: .custom(),
            medias: [],
            isPrivate: false,
            tags: [],
            serverId: resolvedServerId
        )
    }
}
