import SwiftUI
import PinzDomain
import PinzBase

enum AddMediaFlowStatus: String, Equatable {
    case idle
    case sessionStarting
    case uploading
    case grouping
    case ready
    case failed
}

@Observable
final class AddMediaHostViewModel {
    enum Step {
        case selection
        case grouping
        case review
    }

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case backToSelection
        case openReview
        case openGrouping(session: AddMediaStartDTO, loadedMedia: [LoadedMedia])
        case finish
    }

    let tripId: String
    private(set) var step: Step = .selection
    private(set) var flowStatus: AddMediaFlowStatus = .idle
    private(set) var session: AddMediaStartDTO?
    private(set) var loadedMedia: [LoadedMedia] = []
    private(set) var draftPins: [RawPin] = []
    private(set) var existingMediaIds: Set<String> = []
    private(set) var deletedMediaIds: Set<String> = []
    private(set) var existingPinsPreview: [RawPin] = []

    private var router: AppRouting?

    init(tripId: String) {
        self.tripId = tripId
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .backToSelection:
            resetToSelection()
        case .navigate(.back):
            router?.pop()
        case let .openGrouping(session, loadedMedia):
            self.session = session
            self.loadedMedia = loadedMedia
            self.step = .grouping
            self.flowStatus = .uploading
        case .openReview:
            self.step = .review
            self.flowStatus = .ready
        case .finish:
            router?.notifyTripReloadNeeded()
            router?.pop()
        }
    }

    func markGroupingState(
        draftPins: [RawPin],
        existingMediaIds: [String],
        existingPinsPreview: [RawPin]
    ) {
        self.draftPins = draftPins
        self.existingMediaIds = Set(existingMediaIds)
        self.existingPinsPreview = existingPinsPreview
        self.deletedMediaIds = []
        self.flowStatus = .ready
    }

    func markReviewState(
        draftPins: [RawPin],
        existingMediaIds: [String],
        existingPinsPreview: [RawPin],
        deletedMediaIds: [String]
    ) {
        self.draftPins = draftPins
        self.existingMediaIds = Set(existingMediaIds)
        self.existingPinsPreview = existingPinsPreview
        self.deletedMediaIds = Set(deletedMediaIds)
        self.flowStatus = .ready
    }

    func markGroupingFailed() {
        self.flowStatus = .failed
    }

    func resetToSelection() {
        step = .selection
        flowStatus = .idle
        session = nil
        loadedMedia = []
        draftPins = []
        existingMediaIds = []
        deletedMediaIds = []
        existingPinsPreview = []
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
