import Foundation
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
@Observable
public final class PhotoBattleViewModel: Identifiable {

    public static let requiredBattleMediaCount = 8
    public static let requiredBattleComparisons = requiredBattleMediaCount - 1
    public static let totalBattleRounds = 3

    public let id: String
    public let tripId: String
    public let battleSessionId: String

    private let networkService: any NetworkServiceProtocol
    private let onFinish: () -> Void

    public var isPhotoBattlePreloading = false
    public var isSubmittingResult = false
    public var battleError: String?
    public var photoBattleState = PhotoBattleState()
    public var battleMode = PhotoBattleScreenMode.battle
    public var winnerPhotoBattleMedia: PhotoBattleMedia?
    public var winnerBattleRating: Int?

    private var currentRoundMedia: [PhotoBattleMedia] = []
    private var currentRoundWinners: [PhotoBattleMedia] = []
    private var currentPairIndex = 0
    private var isSelectionLocked = false

    public var currentPair: (PhotoBattleMedia, PhotoBattleMedia)? {
        photoBattleState.currentPair
    }

    public var leftMedia: PhotoBattleMedia? {
        photoBattleState.leftMedia
    }

    public var rightMedia: PhotoBattleMedia? {
        photoBattleState.rightMedia
    }

    public var currentRound: Int {
        photoBattleState.currentRound
    }

    public var nextRound: Int {
        photoBattleState.nextRound
    }

    public var step: Int {
        photoBattleState.step
    }

    public var progress: Double {
        photoBattleState.progress
    }

    public var totalBattleSteps: Int {
        Self.requiredBattleComparisons
    }

    public var isBattleControlsBlocked: Bool {
        isSubmittingResult || isSelectionLocked || isPhotoBattlePreloading || battleMode == .winner
    }

    public init(
        tripId: String,
        battleSessionId: String,
        media: [PhotoBattleMedia],
        networkService: any NetworkServiceProtocol,
        onFinish: @escaping () -> Void
    ) {
        self.id = battleSessionId
        self.tripId = tripId
        self.battleSessionId = battleSessionId
        self.networkService = networkService
        self.onFinish = onFinish
        currentRoundMedia = Array(media.prefix(Self.requiredBattleMediaCount))

        photoBattleState = PhotoBattleState(
            isActive: true,
            battleSessionId: battleSessionId,
            currentRound: 1,
            nextRound: 2,
            step: 0,
            progress: 0,
            currentPair: pair(for: 0, in: currentRoundMedia)
        )
        battleMode = .battle
        winnerPhotoBattleMedia = nil
        winnerBattleRating = nil
    }

    public func preloadBattleMedia() async {
        isPhotoBattlePreloading = true
        await preloadBattleMedia(currentRoundMedia)
        if isPhotoBattlePreloading {
            isPhotoBattlePreloading = false
        }
    }

    public func selectPhotoBattleMedia(_ selectedMedia: PhotoBattleMedia) {
        guard !isBattleControlsBlocked else {
            return
        }
        guard let current = photoBattleState.currentPair,
              current.0.photoBattleMediaId == selectedMedia.photoBattleMediaId ||
              current.1.photoBattleMediaId == selectedMedia.photoBattleMediaId else {
            return
        }

        isSelectionLocked = true
        currentRoundWinners.append(selectedMedia)
        photoBattleState.step += 1
        photoBattleState.progress = Double(photoBattleState.step) / Double(totalBattleSteps)

        currentPairIndex += 1
        if let nextPair = pair(for: currentPairIndex, in: currentRoundMedia) {
            photoBattleState.currentPair = nextPair
            isSelectionLocked = false
            return
        }

        if currentRoundWinners.count == 1 {
            let winner = currentRoundWinners[0]
            photoBattleState.currentPair = nil
            Task {
                await finishBattle(with: winner)
            }
            return
        }

        currentRoundMedia = currentRoundWinners
        currentRoundWinners = []
        photoBattleState.currentRound += 1
        photoBattleState.nextRound = min(photoBattleState.currentRound + 1, Self.totalBattleRounds)
        currentPairIndex = 0
        photoBattleState.currentPair = pair(for: currentPairIndex, in: currentRoundMedia)
        isSelectionLocked = false
    }

    public func dismissPhotoBattle() {
        clearBattleState()
        onFinish()
    }

    public func clearPhotoBattleError() {
        battleError = nil
    }

    private func finishBattle(with winner: PhotoBattleMedia) async {
        isSubmittingResult = true
        isSelectionLocked = true
        do {
            let response = try await networkService.submitBattleResult(
                tripId: tripId,
                battleId: battleSessionId,
                winnerMediaId: winner.photoBattleMediaId
            )
            winnerPhotoBattleMedia = winner
            winnerBattleRating = response.newBattleRating
            battleMode = .winner
        } catch {
            battleMode = .battle
            battleError = PinzBaseStrings.PhotoBattle.Error.submitResultFailed
            isSelectionLocked = false
            winnerPhotoBattleMedia = nil
            winnerBattleRating = nil
        }
        isSubmittingResult = false
    }

    private func preloadBattleMedia(_ media: [PhotoBattleMedia]) async {
        await withTaskGroup(of: Void.self) { group in
            for item in media {
                guard let url = item.url?.absoluteString else { continue }

                group.addTask {
                    if item.kind == .video {
                        _ = await ImageProvider.loadOrGetVideoThumbnail(for: url)
                    } else {
                        await ImageProvider.loadAndCacheImage(
                            for: url,
                            .media,
                            cacheVariant: .thumbnail,
                            targetPixel: 560
                        )
                    }
                }
            }

            await group.waitForAll()
        }
    }

    private func pair(for pairIndex: Int, in media: [PhotoBattleMedia]) -> (PhotoBattleMedia, PhotoBattleMedia)? {
        let startIndex = pairIndex * 2
        guard startIndex < media.count else {
            return nil
        }
        guard media.count > startIndex + 1 else {
            return nil
        }
        return (media[startIndex], media[startIndex + 1])
    }

    private func clearBattleState() {
        isPhotoBattlePreloading = false
        isSubmittingResult = false
        isSelectionLocked = false
        battleMode = .battle
        winnerPhotoBattleMedia = nil
        winnerBattleRating = nil
        photoBattleState = PhotoBattleState()
        currentRoundMedia = []
        currentRoundWinners = []
        currentPairIndex = 0
    }
}
