import SwiftUI
import PinzBase
import PinzDomain
import PinzNetworking

@Observable
final class AddMediaReviewViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case confirm
        case cancel
        case takeover
    }

    let tripId: String
    let sessionId: String
    var pins: [Pin] = []
    var canEdit: Bool = false
    var currentInitiator: PublicUserProfileDTO? = nil
    var takeoverAvailableAt: Date? = nil
    private(set) var isLoading = false

    private var wsTask: Task<Void, Never>?
    private var wsClient = AddMediaWebSocketClient()
    private var router: AppRouting?
    private let networkService: NetworkServiceProtocol

    init(tripId: String, sessionId: String, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.networkService = networkService
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
        Task {
            await loadReview()
            startWSListener()
        }
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .navigate(.back):
            router?.pop()
        }
    }

    func navigateToPinInfo(at index: Int) {
        let pin = pins[index]
        router?.navigateToPinInfo(
            pin: pin,
            updateAction: PinUpdateAction { [weak self] updatedPin in
                self?.pins[index] = updatedPin
            },
            deleteAction: nil
        )
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .confirm:
            isLoading = true
            defer { isLoading = false }
            let pinUpdates = pins.map { pin in
                PinUpdateInputDTO(
                    pinId: pin.serverId ?? pin.id,
                    name: pin.name,
                    description: pin.description,
                    category: pin.category.apiValue,
                    privacyLevel: pin.isPrivate ? "private" : "public",
                    latitude: pin.coordinates?.latitude,
                    longitude: pin.coordinates?.longitude,
                    tags: pin.tags.map(\.tag),
                    startTimeUnix: pin.startDate.map { Int($0.timeIntervalSince1970) },
                    endTimeUnix: pin.endDate.map { Int($0.timeIntervalSince1970) }
                )
            }
            _ = try await networkService.addMediaConfirm(
                tripId: tripId,
                sessionId: sessionId,
                pinUpdates: pinUpdates,
                mediaToDelete: []
            )
            router?.popToRoot()

        case .cancel:
            try await networkService.addMediaCancel(tripId: tripId, sessionId: sessionId)
            router?.popToRoot()

        case .takeover:
            isLoading = true
            defer { isLoading = false }
            let dto = try await networkService.addMediaTakeover(tripId: tripId, sessionId: sessionId)
            canEdit = dto.isInitiator
            currentInitiator = dto.currentInitiator
            takeoverAvailableAt = parseDate(dto.takeoverAvailableAt)
        }
    }

    // MARK: - Private

    private func loadReview() async {
        do {
            let dto = try await networkService.addMediaGetReview(tripId: tripId, sessionId: sessionId)
            pins = dto.pins.enumerated().map { index, pinDto in
                pinDto.toPin(
                    index: index,
                    tripId: tripId,
                    nameIfMissing: PinzBaseStrings.Common.Label.pinNumber(index + 1)
                )
            }
            canEdit = dto.canEdit
            currentInitiator = dto.currentInitiator
            takeoverAvailableAt = parseDate(dto.takeoverAvailableAt)
        } catch {
            print("[AddMediaReviewViewModel] Failed to load review: \(error)")
        }
    }

    private func startWSListener() {
        wsTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId) {
                switch event {
                case .initiatorChanged:
                    await loadReview()
                case let .tripStatusChanged(status) where status.uppercased() == "READY":
                    router?.popToRoot()
                    return
                default:
                    break
                }
            }
        }
    }

    private func parseDate(_ string: String?) -> Date? {
        guard let string else { return nil }
        return ISO8601DateFormatter().date(from: string)
    }

    deinit {
        wsTask?.cancel()
        wsClient.disconnect()
    }
}
