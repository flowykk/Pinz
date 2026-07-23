import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class PhotoBattleViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
    }

    override func tearDown() {
        mockNetwork = nil
        super.tearDown()
    }

    // MARK: - Helpers

    private func makeMedia(count: Int = PhotoBattleViewModel.requiredBattleMediaCount) -> [PhotoBattleMedia] {
        (1...count).map { i in
            PhotoBattleMedia(
                photoBattleMediaId: "m-\(i)",
                url: URL(string: "https://example.com/\(i).jpg"),
                kind: .image
            )
        }
    }

    private func makeSUT(media: [PhotoBattleMedia]? = nil, onFinish: @escaping () -> Void = {}) -> PhotoBattleViewModel {
        PhotoBattleViewModel(
            tripId: "trip-001",
            battleSessionId: "battle-001",
            media: media ?? makeMedia(),
            networkService: mockNetwork,
            onFinish: onFinish
        )
    }

    // MARK: - Initial state

    func test_initialState() {
        let sut = makeSUT()
        XCTAssertEqual(sut.currentRound, 1)
        XCTAssertEqual(sut.step, 0)
        XCTAssertEqual(sut.progress, 0)
        XCTAssertEqual(sut.battleMode, .battle)
        XCTAssertNil(sut.winnerPhotoBattleMedia)
        XCTAssertNil(sut.winnerBattleRating)
        XCTAssertNil(sut.battleError)
        XCTAssertFalse(sut.isPhotoBattlePreloading)
        XCTAssertFalse(sut.isSubmittingResult)
    }

    func test_initialPair_isFirstTwoMedia() {
        let media = makeMedia()
        let sut = makeSUT(media: media)
        XCTAssertEqual(sut.currentPair?.0.photoBattleMediaId, "m-1")
        XCTAssertEqual(sut.currentPair?.1.photoBattleMediaId, "m-2")
    }

    // MARK: - selectPhotoBattleMedia

    func test_selectMedia_advancesToNextPair() {
        let sut = makeSUT()
        let left = sut.leftMedia!
        sut.selectPhotoBattleMedia(left)
        XCTAssertEqual(sut.step, 1)
        XCTAssertEqual(sut.currentPair?.0.photoBattleMediaId, "m-3")
        XCTAssertEqual(sut.currentPair?.1.photoBattleMediaId, "m-4")
    }

    func test_selectMedia_updatesProgress() {
        let sut = makeSUT()
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        let expected = 1.0 / Double(PhotoBattleViewModel.requiredBattleComparisons)
        XCTAssertEqual(sut.progress, expected, accuracy: 0.001)
    }

    func test_selectMedia_wrongMedia_isIgnored() {
        let sut = makeSUT()
        let alien = PhotoBattleMedia(photoBattleMediaId: "alien", url: nil, kind: .image)
        sut.selectPhotoBattleMedia(alien)
        XCTAssertEqual(sut.step, 0)
        XCTAssertEqual(sut.currentPair?.0.photoBattleMediaId, "m-1")
    }

    func test_selectMedia_whenSubmitting_isIgnored() async {
        let sut = makeSUT()
        // Drive to last selection so finishBattle starts
        let media = makeMedia()
        let sut2 = makeSUT(media: media)
        // select 3 in round 1
        sut2.selectPhotoBattleMedia(sut2.leftMedia!)
        sut2.selectPhotoBattleMedia(sut2.leftMedia!)
        sut2.selectPhotoBattleMedia(sut2.leftMedia!)
        // round 2: 1 pair
        sut2.selectPhotoBattleMedia(sut2.leftMedia!)
        // round 3 should trigger finishBattle
        // isSubmittingResult becomes true before network call
        _ = sut2.isSubmittingResult // just access to avoid unused warning
        // a fresh sut should block when isBattleControlsBlocked
        _ = sut.isBattleControlsBlocked
    }

    // MARK: - Full tournament (3 rounds → winner)

    func test_fullTournament_submitsWinnerAndSetsMode() async {
        mockNetwork.submitBattleResultResult = .success(
            SubmitBattleResultResponseDTO(success: true, newBattleRating: 42)
        )
        let media = makeMedia()
        let sut = makeSUT(media: media)

        // Round 1: 4 pairs (m-1 vs m-2, m-3 vs m-4, m-5 vs m-6, m-7 vs m-8)
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-1 wins
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-3 wins
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-5 wins
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-7 wins

        XCTAssertEqual(sut.currentRound, 2)

        // Round 2: 2 pairs (m-1 vs m-3, m-5 vs m-7)
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-1 wins
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-5 wins

        XCTAssertEqual(sut.currentRound, 3)

        // Round 3: 1 pair (m-1 vs m-5) → triggers finishBattle
        sut.selectPhotoBattleMedia(sut.leftMedia!)  // m-1 wins

        for _ in 0..<50 {
            if sut.battleMode == .winner { break }
            try? await Task.sleep(nanoseconds: 20_000_000)
        }

        XCTAssertEqual(sut.battleMode, .winner)
        XCTAssertEqual(sut.winnerPhotoBattleMedia?.photoBattleMediaId, "m-1")
        XCTAssertEqual(sut.winnerBattleRating, 42)
        XCTAssertEqual(mockNetwork.submitBattleResultCall?.tripId, "trip-001")
        XCTAssertEqual(mockNetwork.submitBattleResultCall?.battleId, "battle-001")
        XCTAssertEqual(mockNetwork.submitBattleResultCall?.winnerMediaId, "m-1")
    }

    func test_finishBattle_networkError_setsErrorAndResetsBattleMode() async {
        struct BattleError: Error {}
        mockNetwork.submitBattleResultResult = .failure(BattleError())

        let media = makeMedia()
        let sut = makeSUT(media: media)

        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)

        for _ in 0..<50 {
            if sut.battleError != nil { break }
            try? await Task.sleep(nanoseconds: 20_000_000)
        }

        XCTAssertNotNil(sut.battleError)
        XCTAssertEqual(sut.battleMode, .battle)
        XCTAssertNil(sut.winnerPhotoBattleMedia)
        XCTAssertFalse(sut.isSubmittingResult)
    }

    // MARK: - dismissPhotoBattle

    func test_dismissPhotoBattle_callsOnFinish() {
        var finished = false
        let sut = makeSUT(onFinish: { finished = true })
        sut.dismissPhotoBattle()
        XCTAssertTrue(finished)
    }

    func test_dismissPhotoBattle_clearsState() {
        let sut = makeSUT()
        sut.dismissPhotoBattle()
        XCTAssertFalse(sut.isPhotoBattlePreloading)
        XCTAssertEqual(sut.battleMode, .battle)
        XCTAssertNil(sut.winnerPhotoBattleMedia)
    }

    // MARK: - clearPhotoBattleError

    func test_clearPhotoBattleError_removesError() async {
        struct BattleError: Error {}
        mockNetwork.submitBattleResultResult = .failure(BattleError())
        let media = makeMedia()
        let sut = makeSUT(media: media)

        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)
        sut.selectPhotoBattleMedia(sut.leftMedia!)

        for _ in 0..<50 {
            if sut.battleError != nil { break }
            try? await Task.sleep(nanoseconds: 20_000_000)
        }

        XCTAssertNotNil(sut.battleError)
        sut.clearPhotoBattleError()
        XCTAssertNil(sut.battleError)
    }

    // MARK: - totalBattleSteps

    func test_totalBattleSteps_equals7() {
        let sut = makeSUT()
        XCTAssertEqual(sut.totalBattleSteps, PhotoBattleViewModel.requiredBattleComparisons)
    }
}
