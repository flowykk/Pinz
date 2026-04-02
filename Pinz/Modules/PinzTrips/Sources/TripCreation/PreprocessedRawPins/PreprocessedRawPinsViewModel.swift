import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PreprocessedRawPinsViewModel {

    enum Route {
        case back
        case review(tripId: String, pins: [Pin])
    }

    enum Intent {
        case navigate(Route)
        case deleteMedia(RawPinMedia, fromPin: String)
        case mergePins(firstIndex: Int, secondIndex: Int)
        case addPin
        case moveMedia(RawPinMedia, fromPin: Int, toPin: Int)
    }

    enum AsyncIntent {
        case `continue`
    }

    let tripId: String
    var pins: RawPins
    private(set) var isLoading = false

    private let networkService = NetworkService()
    private var router: AppRouting?
    private var deletedMediaIds: [String] = []

    init(tripId: String, pins: RawPins) {
        self.tripId = tripId
        self.pins = pins
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case .review(let tripId, let pins):
                router?.navigateToTripCreationReview(tripId: tripId, pins: pins)
            }
        case let .deleteMedia(media, pinID):
            withAnimation(.easeInOut(duration: 0.3)) {
                if let pinIndex = pins.pins.firstIndex(where: { $0.id == pinID }) {
                    pins.pins[pinIndex].medias.removeAll { $0.id == media.id }
                    deletedMediaIds.append(media.id)
                }
            }
        case let .mergePins(firstIndex, secondIndex):
            guard firstIndex != secondIndex,
                  firstIndex < pins.pins.count,
                  secondIndex < pins.pins.count else { return }
            withAnimation(.easeInOut(duration: 0.3)) {
                pins.pins[firstIndex].medias += pins.pins[secondIndex].medias
                pins.pins.remove(at: secondIndex)
            }
        case .addPin:
            withAnimation(.easeInOut(duration: 0.3)) {
                let newId = UUID().uuidString
                pins.pins.append(RawPin(id: newId, medias: []))
            }
        case let .moveMedia(media, fromPin, toPin):
            withAnimation(.easeInOut(duration: 0.3)) {
                guard fromPin != toPin,
                      fromPin < pins.pins.count,
                      toPin < pins.pins.count else { return }
                pins.pins[fromPin].medias.removeAll { $0.id == media.id }
                pins.pins[toPin].medias.append(media)
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            changeLoading(to: true)
            let draftPins = pins.pins.map { DraftPinInputDTO(draftPinId: $0.id, mediaIds: $0.medias.map(\.id)) }
            try await networkService.applyGroupsAndProcess(
                tripId: tripId,
                draftPins: draftPins,
                deletedMediaIds: deletedMediaIds
            )
            let reviewResponse = try await networkService.getTripReview(tripId: tripId)
            let pins = reviewResponse.pins.enumerated().map { index, dto in dto.toPin(index: index) }
            dispatch(.navigate(.review(tripId: tripId, pins: pins)))
            changeLoading(to: false)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }
}
