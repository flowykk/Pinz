import Foundation
import PinzDomain

public struct PhotoBattleMedia: Identifiable, Hashable {
    public let photoBattleMediaId: String
    public let url: URL?
    public let kind: MediaType

    public init(photoBattleMediaId: String, url: URL?, kind: MediaType) {
        self.photoBattleMediaId = photoBattleMediaId
        self.url = url
        self.kind = kind
    }

    public init(_ dto: StartBattleMediaDTO) {
        photoBattleMediaId = dto.photoBattleMediaId
        url = URL(string: dto.url)
        kind = dto.mediaType.toPhotoBattleKind
    }

    public var id: String { photoBattleMediaId }
}

public enum PhotoBattleScreenMode: Equatable {
    case battle
    case winner
}

public extension String {
    var toPhotoBattleKind: MediaType {
        switch lowercased() {
        case "video": return .video
        default: return .image
        }
    }
}

public struct PhotoBattleState: Equatable {
    public var isActive: Bool
    public var battleSessionId: String?
    public var currentRound: Int
    public var nextRound: Int
    public var step: Int
    public var progress: Double
    public var currentPair: (PhotoBattleMedia, PhotoBattleMedia)?

    public init(
        isActive: Bool = false,
        battleSessionId: String? = nil,
        currentRound: Int = 0,
        nextRound: Int = 0,
        step: Int = 0,
        progress: Double = 0,
        currentPair: (PhotoBattleMedia, PhotoBattleMedia)? = nil
    ) {
        self.isActive = isActive
        self.battleSessionId = battleSessionId
        self.currentRound = currentRound
        self.nextRound = nextRound
        self.step = step
        self.progress = progress
        self.currentPair = currentPair
    }

    public var leftMedia: PhotoBattleMedia? {
        currentPair?.0
    }

    public var rightMedia: PhotoBattleMedia? {
        currentPair?.1
    }

    public static func == (lhs: PhotoBattleState, rhs: PhotoBattleState) -> Bool {
        guard lhs.isActive == rhs.isActive else { return false }
        guard lhs.battleSessionId == rhs.battleSessionId else { return false }
        guard lhs.currentRound == rhs.currentRound else { return false }
        guard lhs.nextRound == rhs.nextRound else { return false }
        guard lhs.step == rhs.step else { return false }
        guard lhs.progress == rhs.progress else { return false }

        switch (lhs.currentPair, rhs.currentPair) {
        case (nil, nil):
            return true
        case let (leftPair?, rightPair?):
            return leftPair.0 == rightPair.0 && leftPair.1 == rightPair.1
        default:
            return false
        }
    }
}
