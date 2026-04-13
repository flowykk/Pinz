import SwiftUI
import Foundation
import PinzDomain
import PinzBase
import PinzNetworking

@MainActor
@Observable
final class TripInfoViewModel {

    enum State {
        case `default`
        case editing
    }

    enum Route {
        case pinsList
        case selectPins
        case back
    }

    enum Intent {
        case changeState
        case setImage(UIImage?)
        case navigate(Route)
    }

    enum AsyncIntent {
        case editTrip
    }

    var state: State = .default

    var trip: Trip
    private var editingSnapshot: Trip?
    private var router: AppRouting?
    private let networkService: any NetworkServiceProtocol
    private let onTripUpdated: (() -> Void)?

    init(trip: Trip, networkService: NetworkServiceProtocol = NetworkService(), onTripUpdated: (() -> Void)? = nil) {
        self.trip = trip
        self.networkService = networkService
        self.onTripUpdated = onTripUpdated
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .changeState:
            switch state {
            case .default:
                backupTripForEditing()
                changeState(to: .editing)
            case .editing:
                restoreTripFromSnapshot()
                changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                trip.image = newImage
            }
        case let .navigate(route):
            switch route {
            case .pinsList:
                router?.navigateToPinsList(trip: trip)
            case .selectPins:
                router?.navigateToSelectablePinsList(trip: trip)
            case .back:
                router?.pop()
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .editTrip:
            try await editTrip()
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    func deleteTrip() async throws {
        try await networkService.deleteTrip(id: trip.id)
        SelectedTripStorage.shared.clearSelection()
    }

    private func editTrip() async throws {
        let updatedTrip = try await networkService.updateTrip(
            id: trip.id,
            name: trip.name,
            description: trip.description,
            category: mapCategory(trip.category),
            season: mapSeason(trip.season),
            privacyLevel: trip.privacyLevel,
            coverUrl: trip.coverUrl,
            startDateUnix: trip.startDate.flatMap { Int($0.timeIntervalSince1970) },
            endDateUnix: trip.endDate.flatMap { Int($0.timeIntervalSince1970) }
        )

        let oldTripImage = trip.image
        let oldPins = trip.pins
        trip = updatedTrip.toTrip()
        trip.image = oldTripImage
        trip.pins = oldPins
        onTripUpdated?()
        changeState(to: .default)
        editingSnapshot = nil
    }

    private func mapCategory(_ category: TripCategory) -> String? {
        switch category {
        case .none:
            nil
        case let .custom(value):
            value
        case .vacation:
            "vacation"
        case .holidays:
            "holidays"
        case .business:
            "business"
        case .education:
            "education"
        case .active:
            "active"
        }
    }

    private func mapSeason(_ season: TripSeason) -> String? {
        switch season {
        case .none:
            nil
        case .summer:
            "summer"
        case .autumn:
            "autumn"
        case .winter:
            "winter"
        case .spring:
            "spring"
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }

    private func backupTripForEditing() {
        editingSnapshot = trip
    }

    private func restoreTripFromSnapshot() {
        if let snapshot = editingSnapshot {
            trip = snapshot
        }
        editingSnapshot = nil
    }

}
