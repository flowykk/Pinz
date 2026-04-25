import SwiftUI
import PinzBase
import PinzDomain
import PinzNetworking

@Observable
final class ReviewTripCreationViewModel {

    enum Route {
        case back
        case problems
    }

    enum Intent {
        case navigate(Route)
    }

    enum AsyncIntent {
        case finalize
    }

    let tripId: String
    var pins: [Pin]

    private var router: AppRouting?
    private let networkService: NetworkServiceProtocol

    var pinsHaveIssues: Bool {
        return pins.contains(where: { !$0.issueKinds.isEmpty })
    }

    init(tripId: String, pins: [Pin], networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.pins = pins
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case .problems:
                syncDraftPins()
                router?.navigateToTripCreationProblems(tripId: tripId, pins: pins)
            }
        }
    }

    func navigateToPinInfo(at index: Int, router: AppRouting?) {
        let pin = pins[index]
        router?.navigateToPinInfo(
            pin: pin,
            updateAction: PinUpdateAction { [weak self] updatedPin in
                var fixedPin = updatedPin
                fixedPin.issues = self?.normalizeIssues(for: updatedPin) ?? []
                self?.pins[index] = fixedPin
                self?.syncDraftPins()
            }
        )
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
        guard let router else { return }

        if let draftPins = router.tripCreationDraftPins(for: tripId) {
            pins = draftPins
        } else {
            router.setTripCreationDraftPins(pins, for: tripId)
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .finalize:
            let pinsForFinalize = router?.tripCreationDraftPins(for: tripId) ?? pins
            let pinUpdates = pinsForFinalize.map { pin -> PinUpdateInputDTO in
                let pinId = pin.serverId ?? pin.id

                return PinUpdateInputDTO(
                    pinId: pinId,
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

            _ = try await networkService.finalizeTrip(
                tripId: tripId,
                pinUpdates: pinUpdates,
                mediaToDelete: []
            )

            router?.clearTripCreationDraftPins(for: tripId)

            router?.pop(by: 3)
        }
    }

    private func normalizeIssues(for pin: Pin) -> [String] {
        var result: [String] = []
        if pin.coordinates == nil {
            result.append(Pin.Issue.missingCoordinates.rawValue)
        }
        if pin.startDate == nil || pin.endDate == nil {
            result.append(Pin.Issue.missingDates.rawValue)
        }
        return result
    }

    private func syncDraftPins() {
        guard let router else { return }
        router.setTripCreationDraftPins(pins, for: tripId)
    }
}
