import XCTest
@testable import PinzUI
import PinzBase
import PinzDomain
import CoreLocation

@MainActor
final class PinProblemsListViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: PinProblemsListViewModel!

    private let tripId = "trip-problems-001"
    private let sessionId = "session-problems-001"

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
    }

    override func tearDown() {
        mockRouter = nil
        sut = nil
        super.tearDown()
    }

    // MARK: - Init (trip creation)

    func test_init_tripCreation_setsPins() {
        let pins = [makePin(name: "Pin1", issues: [])]
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: pins)
        XCTAssertEqual(sut.pins.count, 1)
    }

    // MARK: - pinsWithIssues

    func test_pinsWithIssues_includesOnlyPinsWithIssues() {
        let goodPin = makePin(name: "Good", issues: [])
        let badPin = makePin(name: "Bad", issues: [Pin.Issue.missingCoordinates.rawValue])
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [goodPin, badPin])
        XCTAssertEqual(sut.pinsWithIssues.count, 1)
        XCTAssertEqual(sut.pinsWithIssues[0].pin.name, "Bad")
    }

    func test_pinsWithIssues_empty_whenAllPinsValid() {
        let pins = [makePin(name: "Good", issues: [])]
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: pins)
        XCTAssertTrue(sut.pinsWithIssues.isEmpty)
    }

    func test_pinsWithIssues_includesCorrectPinIndex() {
        let good = makePin(name: "Good", issues: [])
        let bad = makePin(name: "Bad", issues: [Pin.Issue.missingDates.rawValue])
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [good, bad])
        XCTAssertEqual(sut.pinsWithIssues[0].pinIndex, 1)
    }

    func test_pinsWithIssues_multipleIssues_concatenatesText() {
        let pin = makePin(
            name: "Broken",
            issues: [
                Pin.Issue.missingCoordinates.rawValue,
                Pin.Issue.missingDates.rawValue
            ]
        )
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [pin])
        let issueText = sut.pinsWithIssues[0].issueText
        XCTAssertTrue(issueText.contains(", "))
    }

    // MARK: - Navigation

    func test_dispatch_navigate_back_callsPop() {
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [])
        sut.setRouter(mockRouter)
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    // MARK: - setRouter / draft pins (trip creation)

    func test_setRouter_tripCreation_loadsDraftPins_whenAvailable() {
        let draftPins = [makePin(name: "Draft", issues: [])]
        mockRouter.setTripCreationDraftPins(draftPins, for: tripId)
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [makePin(name: "Original", issues: [])])
        sut.setRouter(mockRouter)
        XCTAssertEqual(sut.pins[0].name, "Draft")
    }

    func test_setRouter_tripCreation_savesCurrentPins_whenNoDraftsExist() {
        let pins = [makePin(name: "A", issues: [])]
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: pins)
        sut.setRouter(mockRouter)
        let saved = mockRouter.tripCreationDraftPins(for: tripId)
        XCTAssertNotNil(saved)
        XCTAssertEqual(saved?.first?.name, "A")
    }

    // MARK: - setRouter (pin upload)

    func test_setRouter_pinUpload_loadsDraftPin_whenAvailable() {
        let draft = makePin(name: "UploadDraft", issues: [Pin.Issue.missingDates.rawValue])
        mockRouter.setPinUploadReviewDraftPin(draft, forSessionId: sessionId)
        sut = PinProblemsListViewModel(draftBinding: .pinUpload(sessionId: sessionId), pins: [])
        sut.setRouter(mockRouter)
        XCTAssertEqual(sut.pins.count, 1)
        XCTAssertEqual(sut.pins[0].name, "UploadDraft")
    }

    func test_setRouter_addMediaReview_loadsDraftPins_whenAvailable() {
        let draft = [
            makePin(name: "A", issues: []),
            makePin(name: "B", issues: [Pin.Issue.missingDates.rawValue])
        ]
        mockRouter.setAddMediaReviewDraftPins(draft, forSessionId: sessionId)
        sut = PinProblemsListViewModel(draftBinding: .addMediaReview(sessionId: sessionId), pins: [])
        sut.setRouter(mockRouter)
        XCTAssertEqual(sut.pins.count, 2)
        XCTAssertEqual(sut.pins[1].name, "B")
    }

    // MARK: - navigateToPinInfo

    func test_navigateToPinInfo_callsRouter() {
        let badPin = makePin(name: "Bad", issues: [Pin.Issue.missingCoordinates.rawValue])
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [badPin])
        sut.setRouter(mockRouter)
        sut.navigateToPinInfo(at: 0, router: mockRouter)
        XCTAssertEqual(mockRouter.navigatedPinInfo?.name, "Bad")
    }

    func test_navigateToPinInfo_outOfBounds_doesNotCrash() {
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [])
        sut.setRouter(mockRouter)
        sut.navigateToPinInfo(at: 99, router: mockRouter)
        XCTAssertNil(mockRouter.navigatedPinInfo)
    }

    func test_navigateToPinInfo_updateAction_fixesCoordinatesIssue() {
        let badPin = makePin(name: "NoCoord", issues: [Pin.Issue.missingCoordinates.rawValue])
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [badPin])
        sut.setRouter(mockRouter)
        sut.navigateToPinInfo(at: 0, router: mockRouter)

        let fixed = makePin(
            name: "NoCoord",
            issues: [],
            coordinates: CLLocationCoordinate2D(latitude: 55.0, longitude: 37.0),
            startDate: Date(),
            endDate: Date()
        )
        mockRouter.navigatedPinUpdateAction?.action(fixed)

        XCTAssertTrue(sut.pins[0].issueKinds.isEmpty)
    }

    func test_navigateToPinInfo_updateAction_addsCoordinatesIssue_whenCoordsMissing() {
        let pin = makePin(name: "Pin", issues: [])
        sut = PinProblemsListViewModel(draftBinding: .tripCreation(tripId: tripId), pins: [pin])
        sut.setRouter(mockRouter)

        let badPin = makePin(name: "Bad", issues: [Pin.Issue.missingCoordinates.rawValue])
        sut.pins = [badPin]
        sut.navigateToPinInfo(at: 0, router: mockRouter)

        let stillBroken = makePin(name: "Bad", issues: [], coordinates: nil, startDate: Date(), endDate: Date())
        mockRouter.navigatedPinUpdateAction?.action(stillBroken)

        XCTAssertTrue(sut.pins[0].issueKinds.contains(.missingCoordinates))
    }

    // MARK: - Helpers

    private func makePin(
        name: String = "Test",
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
