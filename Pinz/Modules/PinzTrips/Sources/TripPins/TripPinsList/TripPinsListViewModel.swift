import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class TripPinsListViewModel {

    enum Route {
        case pinInfo(Pin)
        case pinCreation
        case addMedia
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case addMedia
    }

    var trip: Trip

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    init(trip: Trip) {
        self.trip = trip
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .pinInfo(pin):
                router?.navigateToPinInfo(
                    pin: pin,
                    updateAction: PinUpdateAction { [weak self] updatedPin in
                        guard let self, let idx = trip.pins.firstIndex(where: { $0.serverId == updatedPin.serverId }) else { return }
                        trip.pins[idx] = updatedPin
                    },
                    deleteAction: PinDeleteAction { [weak self] deletedPin in
                        guard let self else { return }
                        trip.pins.removeAll { $0.serverId == deletedPin.serverId }
                    }
                )
            case .pinCreation:
                router?.navigateToPinCreation()
            case .addMedia:
                router?.navigateToAddMediaStart(tripId: trip.id)
            case .back:
                router?.pop()
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .addMedia:
            let response = try await networkService.getTrip(id: trip.id)
            let sessionId = response.activeAddMediaSession?.sessionId
            switch response.trip.status ?? "" {
            case "READY":
                dispatch(.navigate(.addMedia))
            case "ADD_MEDIA_UPLOADING":
                if let sessionId { router?.navigateToAddMediaUploading(tripId: trip.id, sessionId: sessionId) }
            case "ADD_MEDIA_GROUPING_REVIEW":
                if let sessionId { router?.navigateToAddMediaGrouping(tripId: trip.id, sessionId: sessionId) }
            case "ADD_MEDIA_PROCESSING":
                if let sessionId { router?.navigateToAddMediaProcessing(tripId: trip.id, sessionId: sessionId) }
            case "ADD_MEDIA_DRAFT_FINAL_REVIEW":
                if let sessionId { router?.navigateToAddMediaReview(tripId: trip.id, sessionId: sessionId) }
            default:
                break
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
