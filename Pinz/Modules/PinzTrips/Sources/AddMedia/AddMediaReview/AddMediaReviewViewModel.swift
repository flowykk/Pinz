import SwiftUI
import PinzBase
import PinzNetworking
import PinzDomain

@MainActor @Observable
final class AddMediaReviewViewModel {
    enum FlowStatus {
        case ready
        case applying
        case failed
    }

    enum LoadingStatus {
        case applying

        var localizedValue: String {
            switch self {
            case .applying:
                PinzBaseStrings.AddMedia.Loading.applying
            }
        }
    }

    enum Intent {
        case deleteMedia(RawPinMedia, fromPin: String)
        case moveMedia(RawPinMedia, fromPin: Int, toPin: Int)
    }

    enum AsyncIntent {
        case apply
    }

    let tripId: String
    let session: AddMediaStartDTO
    private(set) var isLoading = false
    private(set) var loadingStatus: LoadingStatus?
    private(set) var hasFailed = false
    private(set) var flowStatus: FlowStatus = .ready
    private(set) var draftPins: [RawPin]
    private(set) var existingMediaIds: Set<String>
    private(set) var existingPinsPreview: [RawPin]
    private(set) var deletedMediaIds: Set<String>

    private let networkService = NetworkService.shared

    init(
        tripId: String,
        session: AddMediaStartDTO,
        draftPins: [RawPin],
        existingMediaIds: [String],
        existingPinsPreview: [RawPin],
        deletedMediaIds: [String]
    ) {
        self.tripId = tripId
        self.session = session
        self.draftPins = draftPins
        self.existingMediaIds = Set(existingMediaIds)
        self.existingPinsPreview = existingPinsPreview
        self.deletedMediaIds = Set(deletedMediaIds)
    }

    var canApply: Bool {
        draftPins.contains {
            $0.medias.contains { !existingMediaIds.contains($0.id) }
        }
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .deleteMedia(media, pinId):
            guard !existingMediaIds.contains(media.id) else {
                return
            }
            withAnimation(.easeInOut(duration: 0.3)) {
                if let pinIndex = draftPins.firstIndex(where: { $0.id == pinId }) {
                    draftPins[pinIndex].medias.removeAll { $0.id == media.id }
                    deletedMediaIds.insert(media.id)
                }
            }
        case let .moveMedia(media, fromPin, toPin):
            guard !existingMediaIds.contains(media.id),
                  fromPin != toPin,
                  fromPin < draftPins.count,
                  toPin < draftPins.count else {
                return
            }

            withAnimation(.easeInOut(duration: 0.3)) {
                draftPins[fromPin].medias.removeAll { $0.id == media.id }
                draftPins[toPin].medias.append(media)
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .apply:
            try await apply()
        }
    }

    func markFailed() {
        hasFailed = true
        flowStatus = .failed
    }

    private func apply() async throws {
        guard canApply else {
            return
        }

        flowStatus = .applying
        isLoading = true
        loadingStatus = .applying
        hasFailed = false

        do {
            let draftPinsRequest = buildDraftPinsRequest()
            _ = try await networkService.addMediaApplyGroupsAndProcess(
                tripId: tripId,
                sessionId: session.sessionId,
                draftPins: draftPinsRequest,
                deletedMediaIds: Array(deletedMediaIds)
            )

            isLoading = false
            loadingStatus = nil
            flowStatus = .ready
        } catch {
            isLoading = false
            loadingStatus = nil
            flowStatus = .failed
            throw error
        }
    }

    private func buildDraftPinsRequest() -> [DraftPinInputDTO] {
        draftPins.compactMap { pin in
            let mediaIds = pin.medias
                .map(\.id)
                .filter { !existingMediaIds.contains($0) }

            guard !mediaIds.isEmpty else {
                return nil
            }
            return DraftPinInputDTO(draftPinId: pin.id, mediaIds: mediaIds)
        }
    }
}
