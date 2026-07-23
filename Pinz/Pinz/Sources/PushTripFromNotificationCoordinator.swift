import Foundation
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class PushTripFromNotificationCoordinator {
    static let shared = PushTripFromNotificationCoordinator()

    weak var router: AppRouting?
    var showToast: ((String) -> Void)?

    private let networkService: NetworkServiceProtocol

    init(networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.networkService = networkService
    }

    func handle(userInfo: [AnyHashable: Any]) async {
        guard TokenStorage.shared.isAuthenticated else { return }
        guard let tripId = userInfo["trip_id"] as? String, !tripId.isEmpty else { return }
        guard let router else { return }

        do {
            let response = try await networkService.getTrip(id: tripId)
            let trip = await loadTripForNavigation(from: response)
            SelectedTripStorage.shared.selectTrip(id: trip.id)
            router.popToRoot()
            router.navigateToTripInfo(trip: trip, onTripUpdated: nil)
        } catch {
            showToast?(PinzBaseStrings.TripMembers.Invite.deepLinkJoinFailed)
        }
    }

    private func loadTripForNavigation(from response: GetTripResponseDTO) async -> Trip {
        var trip = response.trip.toTrip()
        if let coverUrl = trip.coverUrl {
            trip.image = await ImageProvider.loadOrGetImage(
                for: coverUrl,
                .group,
                cacheVariant: .thumbnail,
                targetPixel: 560
            )
        }
        trip.pins = response.pins.enumerated().map { index, dto in
            dto.toPin(
                index: index,
                tripId: trip.id,
                nameIfMissing: PinzBaseStrings.Common.Label.pinNumber(index + 1)
            )
        }
        return trip
    }
}
