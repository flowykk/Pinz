import XCTest
@testable import PinzPins
import PinzBase
import PinzNetworking

@MainActor
final class PinUploadEntryResolverTests: XCTestCase {

    private let tripId = "trip-resume-test"
    private let pinId = "pin-resume-test"
    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
        PinUploadSessionStorage.shared.clear(forTripId: tripId)
    }

    override func tearDown() {
        PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
        PinUploadSessionStorage.shared.clear(forTripId: tripId)
        mockNetwork = nil
        mockRouter = nil
        super.tearDown()
    }

    func test_resumeAddition_noStoredSession_navigatesToStartWithTargetPinId() async {
        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: nil
        )

        XCTAssertEqual(mockRouter.navigatedToPinUploadStart?.tripId, tripId)
        XCTAssertEqual(mockRouter.navigatedToPinUploadStart?.targetPinId, pinId)
        XCTAssertNil(mockRouter.navigatedToPinUploadProcessing)
        XCTAssertNil(mockRouter.navigatedToPinUploadReview)
    }

    func test_resumeAddition_readyForReview_navigatesToReviewWithTargetPinId() async {
        PinUploadAdditionSessionStorage.shared.save(sessionId: "sess-1", tripId: tripId, pinId: pinId)
        mockNetwork.pinUploadGetReviewResult = .success(
            PinUploadReviewResponseDTO(
                sessionId: "sess-1",
                processingStatus: "READY_FOR_REVIEW",
                draft: nil,
                similar: nil
            )
        )

        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: nil
        )

        XCTAssertEqual(mockRouter.navigatedToPinUploadReview?.tripId, tripId)
        XCTAssertEqual(mockRouter.navigatedToPinUploadReview?.sessionId, "sess-1")
        XCTAssertEqual(mockRouter.navigatedToPinUploadReview?.targetPinId, pinId)
        XCTAssertNil(mockRouter.navigatedToPinUploadStart)
    }

    func test_resumeAddition_processing_navigatesToProcessingWithTargetPinId() async {
        PinUploadAdditionSessionStorage.shared.save(sessionId: "sess-2", tripId: tripId, pinId: pinId)
        mockNetwork.pinUploadGetReviewResult = .success(
            PinUploadReviewResponseDTO(
                sessionId: "sess-2",
                processingStatus: "PROCESSING",
                draft: nil,
                similar: nil
            )
        )

        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: nil
        )

        XCTAssertEqual(mockRouter.navigatedToPinUploadProcessing?.tripId, tripId)
        XCTAssertEqual(mockRouter.navigatedToPinUploadProcessing?.sessionId, "sess-2")
        XCTAssertEqual(mockRouter.navigatedToPinUploadProcessing?.targetPinId, pinId)
    }

    func test_resumeAddition_unknownStatus_clearsStorageAndStartsOver() async {
        PinUploadAdditionSessionStorage.shared.save(sessionId: "sess-dead", tripId: tripId, pinId: pinId)
        mockNetwork.pinUploadGetReviewResult = .success(
            PinUploadReviewResponseDTO(
                sessionId: "sess-dead",
                processingStatus: "CLOSED",
                draft: nil,
                similar: nil
            )
        )

        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: nil
        )

        XCTAssertNil(PinUploadAdditionSessionStorage.shared.sessionId(tripId: tripId, pinId: pinId))
        XCTAssertEqual(mockRouter.navigatedToPinUploadStart?.tripId, tripId)
        XCTAssertEqual(mockRouter.navigatedToPinUploadStart?.targetPinId, pinId)
    }

    func test_resumeAddition_emptyPinId_doesNothing() async {
        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: "   ",
            networkService: mockNetwork,
            router: mockRouter,
            showToast: nil
        )

        XCTAssertNil(mockRouter.navigatedToPinUploadStart)
    }

    func test_resumeAddition_genericError_showsToastAndKeepsStorage() async {
        PinUploadAdditionSessionStorage.shared.save(sessionId: "sess-err", tripId: tripId, pinId: pinId)
        mockNetwork.pinUploadGetReviewResult = .failure(NSError(domain: "test", code: -1))

        var toasts: [String] = []
        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: { toasts.append($0) }
        )

        XCTAssertEqual(toasts, [PinzBaseStrings.PinUpload.Addition.Resume.restoreFailed])
        XCTAssertNotNil(PinUploadAdditionSessionStorage.shared.sessionId(tripId: tripId, pinId: pinId))
        XCTAssertNil(mockRouter.navigatedToPinUploadStart)
    }

    func test_resume_creation_conflict_usesLocalizedToast() async {
        PinUploadSessionStorage.shared.save(sessionId: "sess-c", forTripId: tripId)
        mockNetwork.pinUploadGetReviewResult = .failure(HTTPError.conflict)

        var toasts: [String] = []
        await PinUploadEntryResolver.resume(
            tripId: tripId,
            networkService: mockNetwork,
            router: mockRouter,
            showToast: { toasts.append($0) }
        )

        XCTAssertEqual(toasts, [PinzBaseStrings.PinUpload.Creation.Resume.conflictSession])
        XCTAssertNil(PinUploadSessionStorage.shared.sessionId(forTripId: tripId))
    }
}
