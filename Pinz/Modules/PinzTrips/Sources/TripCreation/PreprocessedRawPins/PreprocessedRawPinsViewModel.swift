import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PreprocessedRawPinsViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case deleteMedia(RawPinMedia, fromPin: UUID)
        case mergePins(firstIndex: Int, secondIndex: Int)
        case addPin
        case moveMedia(RawPinMedia, fromPin: Int, toPin: Int)
    }

    var pins: RawPins

    private let networkService = NetworkService()
    private var router: AppRouting?

    init(pins: RawPins) {
        self.pins = pins
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .deleteMedia(media, pinID):
            if let pinIndex = pins.pins.firstIndex(where: { $0.id == pinID }) {
                pins.pins[pinIndex].medias.removeAll { $0.id == media.id }
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
                pins.pins.append(RawPin(medias: []))
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

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
