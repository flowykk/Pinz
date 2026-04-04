import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class TripsListPopupViewModel {

    var trips: [Trip] = []
    private(set) var isLoading = false

    private let networkService = NetworkService()

    enum AsyncIntent {
        case fetchTrips(selectedTripId: String)
    }

    init() {
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .fetchTrips(let selectedTripId):
            changeLoading(to: true)
            let dtos = try await networkService.getTrips()
            trips = dtos
                .map { $0.toTrip() }
                .filter { $0.id != selectedTripId }
            changeLoading(to: false)
        }
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }
}
