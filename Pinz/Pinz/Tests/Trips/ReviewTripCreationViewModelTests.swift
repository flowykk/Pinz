import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking
import CoreLocation

final class ReviewTripCreationViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: ReviewTripCreationViewModel!

    private let tripId = "trip-review-001"
    private var pins: [Pin] = []

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        pins = Pin.stubs()
        sut = ReviewTripCreationViewModel(tripId: tripId, pins: pins, networkService: mockNetwork)
    }

    override func tearDown() {
        mockRouter = nil
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    // MARK: - Init

    func test_init_setsProperties() {
        XCTAssertEqual(sut.tripId, tripId)
        XCTAssertEqual(sut.pins.count, pins.count)
    }

    // MARK: - pinsHaveIssues

    func test_pinsHaveIssues_false_whenNoPinsHaveIssues() {
        sut = ReviewTripCreationViewModel(tripId: tripId, pins: [makePin(issues: [])], networkService: mockNetwork)
        XCTAssertFalse(sut.pinsHaveIssues)
    }

    func test_pinsHaveIssues_true_whenAnyPinHasIssue() {
        sut = ReviewTripCreationViewModel(
            tripId: tripId,
            pins: [makePin(issues: [Pin.Issue.missingCoordinates.rawValue])],
            networkService: mockNetwork
        )
        XCTAssertTrue(sut.pinsHaveIssues)
    }

    // MARK: - Navigation

    func test_dispatch_navigate_back_callsPop() {
        sut.setRouter(mockRouter)
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_dispatch_navigate_problems_callsRouter() {
        sut.setRouter(mockRouter)
        sut.dispatch(.navigate(.problems))
        XCTAssertEqual(mockRouter.navigatedTripCreationProblems?.tripId, tripId)
    }

    // MARK: - setRouter / draft pins

    func test_setRouter_loadsDraftPinsFromRouter_whenAvailable() {
        let draftPins = [makePin(name: "Draft Pin", issues: [])]
        mockRouter.setTripCreationDraftPins(draftPins, for: tripId)
        sut.setRouter(mockRouter)
        XCTAssertEqual(sut.pins.count, draftPins.count)
        XCTAssertEqual(sut.pins[0].name, "Draft Pin")
    }

    func test_setRouter_savesPinsAsDrafts_whenNoneExist() {
        sut = ReviewTripCreationViewModel(tripId: tripId, pins: pins, networkService: mockNetwork)
        sut.setRouter(mockRouter)
        let saved = mockRouter.tripCreationDraftPins(for: tripId)
        XCTAssertNotNil(saved)
        XCTAssertEqual(saved?.count, pins.count)
    }

    // MARK: - asyncDispatch finalize

    func test_asyncDispatch_finalize_success_popsBy3() async throws {
        sut.setRouter(mockRouter)
        mockNetwork.finalizeTripResult = .success(
            FinalizeTripDTO(tripId: tripId, status: "finalized", message: "done")
        )
        try await sut.asyncDispatch(.finalize)
        XCTAssertEqual(mockRouter.lastPopByCount, 3)
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_asyncDispatch_finalize_clearsDraftPins() async throws {
        sut.setRouter(mockRouter)
        mockRouter.setTripCreationDraftPins(pins, for: tripId)
        mockNetwork.finalizeTripResult = .success(
            FinalizeTripDTO(tripId: tripId, status: "finalized", message: "done")
        )
        try await sut.asyncDispatch(.finalize)
        XCTAssertNil(mockRouter.tripCreationDraftPins(for: tripId))
    }

    func test_asyncDispatch_finalize_failure_throws() async {
        sut.setRouter(mockRouter)
        struct FinalizeError: Error {}
        mockNetwork.finalizeTripResult = .failure(FinalizeError())
        do {
            try await sut.asyncDispatch(.finalize)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is FinalizeError)
        }
    }

    // MARK: - navigateToPinInfo update action

    func test_navigateToPinInfo_updateAction_updatesPin() {
        sut.setRouter(mockRouter)
        let initialPin = makePin(name: "Old", issues: [Pin.Issue.missingCoordinates.rawValue])
        sut = ReviewTripCreationViewModel(tripId: tripId, pins: [initialPin], networkService: mockNetwork)
        sut.setRouter(mockRouter)

        sut.navigateToPinInfo(at: 0, router: mockRouter)

        let updatedPin = makePin(
            name: "Old",
            issues: [],
            coordinates: CLLocationCoordinate2D(latitude: 55.0, longitude: 37.0),
            startDate: Date(),
            endDate: Date()
        )
        mockRouter.navigatedPinUpdateAction?.action(updatedPin)

        XCTAssertEqual(sut.pins[0].name, "Old")
        XCTAssertTrue(sut.pins[0].issueKinds.isEmpty)
    }

    // MARK: - Helpers

    private func makePin(
        name: String = "Test Pin",
        issues: [String] = [],
        coordinates: CLLocationCoordinate2D? = nil,
        startDate: Date? = nil,
        endDate: Date? = nil
    ) -> Pin {
        Pin(
            name: name,
            description: nil,
            category: .entertainment,
            medias: [],
            isPrivate: false,
            startDate: startDate,
            endDate: endDate,
            tags: [],
            issues: issues,
            serverId: UUID().uuidString,
            coordinates: coordinates
        )
    }
}
