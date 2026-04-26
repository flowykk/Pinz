import SwiftUI
import MapKit
import PinzNetworking
import PinzBase
import PinzDomain

@Observable
final class PostPreviewViewModel {

    enum Route {
        case pinInfo(Pin)
        case back(by: Int = 1)
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case publish
    }

    var trip: Trip
    let selectedPins: [Pin]
    var position: MapCameraPosition

    init(
        trip: Trip,
        selectedPins: [Pin],
        networkService: NetworkServiceProtocol = NetworkService.shared
    ) {
        self.trip = trip
        self.selectedPins = selectedPins.map { pin in
            Pin(
                name: pin.name,
                description: pin.description,
                category: pin.category,
                medias: pin.medias.filter { !$0.isPrivate },
                isPrivate: pin.isPrivate,
                startDate: pin.startDate,
                endDate: pin.endDate,
                tags: pin.tags,
                issues: pin.issues,
            coordinates: pin.coordinates
            )
        }
        self.position = self.selectedPins.calculateInitialMapPosition()
        self.networkService = networkService
    }

    var isPublishing: Bool = false
    var publishError: String?

    private let networkService: NetworkServiceProtocol
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case let .pinInfo(pin):
                router?.navigateToPinInfo(pin: pin, updateAction: nil, deleteAction: nil)
            case let .back(depth):
                router?.pop(by: depth)
            }
        }
    }

    func asyncDispatch(
        _ intent: AsyncIntent,
        onError: ((Error) -> Void)? = nil
    ) async {
        do {
            try await executeAsyncIntent(intent)
        } catch {
            publishError = error.localizedDescription
            onError?(error)
        }
    }

    private func executeAsyncIntent(_ intent: AsyncIntent) async throws {
        switch intent {
        case .publish:
            try await publishTrip()
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func publishTrip() async throws {
        isPublishing = true
        publishError = nil
        defer { isPublishing = false }

        let normalizedPinIds = selectedPins
            .map { $0.serverId ?? $0.id }
            .filter { !$0.isEmpty }
            .reduce(into: [String]()) { result, id in
                if !result.contains(id) {
                    result.append(id)
                }
            }

        let publishWhole = selectedPins.count == trip.pins.count

        let response = try await networkService.publishTrip(
            id: trip.id,
            publishWhole: publishWhole,
            pinIds: normalizedPinIds
        )
        let published = response.toTrip()
        trip.status = published.status
        trip.isPublished = published.isPublished
        trip.updatedAt = published.updatedAt

        router?.pop(by: 2)
    }
}
