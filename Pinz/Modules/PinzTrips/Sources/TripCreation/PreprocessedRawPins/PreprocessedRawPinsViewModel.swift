import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PreprocessedRawPinsViewModel {

    enum Route {
        case back
        case review
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
            case .review:
                router?.navigateToTripCreationReview()
            }
        case let .deleteMedia(media, pinID):
            withAnimation(.easeInOut(duration: 0.3)) {
                if let pinIndex = pins.pins.firstIndex(where: { $0.id == pinID }) {
                    pins.pins[pinIndex].medias.removeAll { $0.id == media.id }
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
            try await Task.sleep(nanoseconds: 1_000_000_000)
            dispatch(.navigate(.review))
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
