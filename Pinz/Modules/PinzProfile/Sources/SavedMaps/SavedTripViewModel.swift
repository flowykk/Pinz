import SwiftUI
import MapKit
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class SavedTripViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    private(set) var trip: Trip
    private(set) var pins: [Pin] = []
    var position: MapCameraPosition
    var isLoading = true
    var loadError: String?
    var isSaved = true
    var isTogglingSaved = false

    private let networkService: NetworkServiceProtocol
    private var router: AppRouting?

    init(
        trip: Trip,
        networkService: NetworkServiceProtocol = NetworkService.shared
    ) {
        self.trip = trip
        self.pins = trip.pins
        self.position = trip.pins.calculateInitialMapPosition(
            zoomMultiplier: 2.5,
            topOffsetFactor: 0.2
        )
        self.networkService = networkService
    }

    func loadTrip() async {
        isLoading = true
        loadError = nil
        do {
            let response = try await networkService.getTrip(id: trip.id)
            var loaded = response.trip.toTrip()
            if let coverUrl = loaded.coverUrl {
                loaded.image = await ImageProvider.loadOrGetImage(
                    for: coverUrl,
                    .group
                )
            }
            let mappedPins: [Pin] = response.pins.enumerated().map { index, dto in
                dto.toPin(
                    index: index,
                    tripId: loaded.id,
                    nameIfMissing: PinzBaseStrings.Common.Label.pinNumber(index + 1)
                )
            }
            trip = loaded
            trip.pins = mappedPins
            pins = mappedPins
            position = mappedPins.calculateInitialMapPosition(
                zoomMultiplier: 2.5,
                topOffsetFactor: 0.2
            )
        } catch {
            loadError = error.localizedDescription
        }
        isLoading = false
    }

    func toggleSaved() async {
        guard !isTogglingSaved else { return }
        isTogglingSaved = true
        defer { isTogglingSaved = false }
        if isSaved {
            do {
                try await networkService.removeTripFromFavourites(id: trip.id)
                isSaved = false
            } catch {
                // Keep isSaved; optional surface in UI
            }
        } else {
            do {
                _ = try await networkService.addTripToFavourites(id: trip.id)
                isSaved = true
            } catch {
            }
        }
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
