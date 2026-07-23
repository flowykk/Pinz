import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class AddMediaGroupingViewModel {

    enum Route {
        case review(tripId: String, sessionId: String)
        case uploading(tripId: String, sessionId: String)
        case back
    }

    enum Intent {
        case navigate(Route)
        case deleteMedia(RawPinMedia, fromPin: String)
        case moveMedia(RawPinMedia, fromPin: Int, toPin: Int)
        case mergePins(firstIndex: Int, secondIndex: Int)
        case addPin
    }

    enum AsyncIntent {
        case apply
        case addMore
        case cancel
    }

    let tripId: String
    let sessionId: String
    var rawPins: RawPins = RawPins(pins: [])
    private(set) var isLoading = false
    private var deletedMediaIds: [String] = []

    private var wsTask: Task<Void, Never>?
    private var reviewPollTask: Task<Void, Never>?
    private var wsClient = AddMediaWebSocketClient()
    private var router: AppRouting?
    private let networkService: NetworkServiceProtocol
    private var didNavigateToAddMediaReview = false

    init(tripId: String, sessionId: String, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.networkService = networkService
    }

    // MARK: - dispatch

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case let .review(tripId, sessionId):
                router?.navigateToAddMediaReview(tripId: tripId, sessionId: sessionId)
            case let .uploading(tripId, sessionId):
                router?.navigateToAddMediaUploading(tripId: tripId, sessionId: sessionId)
            }

        case let .deleteMedia(media, pinId):
            withAnimation(.easeInOut(duration: 0.3)) {
                guard let pinIndex = rawPins.pins.firstIndex(where: { $0.id == pinId }) else { return }
                rawPins.pins[pinIndex].medias.removeAll { $0.id == media.id }
                deletedMediaIds.append(media.id)
            }

        case let .moveMedia(media, fromPin, toPin):
            withAnimation(.easeInOut(duration: 0.3)) {
                guard fromPin != toPin,
                      fromPin < rawPins.pins.count,
                      toPin < rawPins.pins.count else { return }
                rawPins.pins[fromPin].medias.removeAll { $0.id == media.id }
                rawPins.pins[toPin].medias.append(media)
            }

        case let .mergePins(firstIndex, secondIndex):
            guard firstIndex != secondIndex,
                  firstIndex < rawPins.pins.count,
                  secondIndex < rawPins.pins.count else { return }
            withAnimation(.easeInOut(duration: 0.3)) {
                rawPins.pins[firstIndex].medias += rawPins.pins[secondIndex].medias
                rawPins.pins.remove(at: secondIndex)
            }

        case .addPin:
            withAnimation(.easeInOut(duration: 0.3)) {
                rawPins.pins.append(RawPin(id: UUID().uuidString, medias: []))
            }
        }
    }

    // MARK: - asyncDispatch

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .apply:
            isLoading = true
            let draftPins = rawPins.pins.map {
                DraftPinInputDTO(draftPinId: $0.id, mediaIds: $0.medias.map(\.id))
            }
            do {
                let applied = try await networkService.addMediaApplyGroupsAndProcess(
                    tripId: tripId,
                    sessionId: sessionId,
                    draftPins: draftPins,
                    deletedMediaIds: deletedMediaIds
                )
                if normalizedAddMediaTripStatus(applied.status) == normalizedAddMediaTripStatus("ADD_MEDIA_DRAFT_FINAL_REVIEW") {
                    navigateToReviewFromGroupingIfNeeded()
                } else if normalizedAddMediaTripStatus(applied.status) == normalizedAddMediaTripStatus("ADD_MEDIA_PROCESSING") {
                    // Прод отдаёт 202 + PROCESSING; WS у тебя падает с -1011 — без poll остаёмся на лоадере.
                    print("[AddMediaGroupingViewModel] apply status=PROCESSING, start getReview poll fallback")
                    reviewPollTask = Task { [weak self] in
                        await self?.pollAddMediaReviewUntilReady()
                    }
                }
                // иначе ждём только WS (редкие статусы)
            } catch {
                isLoading = false
                throw error
            }

        case .addMore:
            isLoading = true
            defer { isLoading = false }
            _ = try await networkService.addMediaProcessGrouping(tripId: tripId, sessionId: sessionId, addMore: true)
            dispatch(.navigate(.uploading(tripId: tripId, sessionId: sessionId)))

        case .cancel:
            try await networkService.addMediaCancel(tripId: tripId, sessionId: sessionId)
            router?.popToRoot()
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
        Task {
            do {
                let dto = try await networkService.addMediaGetGrouping(tripId: tripId, sessionId: sessionId)
                rawPins = toRawPins(dto.draftPins)
            } catch {
                print("[AddMediaGroupingViewModel] failed to load grouping: \(error)")
            }
            startWSListener()
        }
    }

    // MARK: - Private

    private func toRawPins(_ dtos: [DraftPinDTO]) -> RawPins {
        RawPins(pins: dtos.map { pin in
            RawPin(id: pin.draftPinId, medias: pin.media.map { m in
                RawPinMedia(id: m.mediaId, url: m.url, type: m.type == "video" ? .video : .image)
            })
        })
    }

    private func startWSListener() {
        wsTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId) {
                guard case let .tripStatusChanged(status) = event else { continue }
                switch status.uppercased() {
                case "ADD_MEDIA_PROCESSING":
                    isLoading = true
                case "ADD_MEDIA_DRAFT_FINAL_REVIEW":
                    navigateToReviewFromGroupingIfNeeded()
                    return
                default:
                    break
                }
            }
        }
    }

    private func navigateToReviewFromGroupingIfNeeded() {
        guard !didNavigateToAddMediaReview else { return }
        didNavigateToAddMediaReview = true
        reviewPollTask?.cancel()
        reviewPollTask = nil
        wsTask?.cancel()
        wsClient.disconnect()
        isLoading = false
        dispatch(.navigate(.review(tripId: tripId, sessionId: sessionId)))
    }

    private func pollAddMediaReviewUntilReady() async {
        let maxAttempts = 60
        for attempt in 1...maxAttempts {
            try? await Task.sleep(nanoseconds: 1_500_000_000)
            guard !Task.isCancelled else {
                print("[AddMediaGroupingViewModel] poll review cancelled")
                return
            }
            guard !didNavigateToAddMediaReview else { return }
            do {
                _ = try await networkService.addMediaGetReview(tripId: tripId, sessionId: sessionId)
                print("[AddMediaGroupingViewModel] poll getReview OK attempt \(attempt)/\(maxAttempts)")
                navigateToReviewFromGroupingIfNeeded()
                return
            } catch {
                if attempt <= 3 || attempt % 5 == 0 {
                    print("[AddMediaGroupingViewModel] poll getReview (\(attempt)/\(maxAttempts)): \(error.localizedDescription)")
                }
            }
        }
        print("[AddMediaGroupingViewModel] poll getReview timed out")
        if !didNavigateToAddMediaReview {
            isLoading = false
        }
    }

    deinit {
        reviewPollTask?.cancel()
        wsTask?.cancel()
        wsClient.disconnect()
    }

    /// Как `AddMediaWebSocketClient` при разборе `TRIP_STATUS_CHANGED` — чтобы HTTP-ответ apply совпадал с WS.
    private func normalizedAddMediaTripStatus(_ raw: String) -> String {
        raw.uppercased()
            .replacingOccurrences(of: "_", with: "")
            .replacingOccurrences(of: " ", with: "")
    }
}
