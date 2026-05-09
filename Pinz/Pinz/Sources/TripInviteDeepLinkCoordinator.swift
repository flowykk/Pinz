import Foundation
import PinzBase
import PinzDomain
import PinzNavigation
import PinzNetworking

@MainActor
final class TripInviteDeepLinkCoordinator {

    private let router: AppRouter
    private let networkService: NetworkServiceProtocol
    private let showToast: (String) -> Void

    init(
        router: AppRouter,
        networkService: NetworkServiceProtocol = NetworkService.shared,
        showToast: @escaping (String) -> Void
    ) {
        self.router = router
        self.networkService = networkService
        self.showToast = showToast
    }

    func handleIncomingURL(_ url: URL) async {
        guard let token = TripInviteLinkParser.inviteToken(from: url) else { return }
        guard TokenStorage.shared.isAuthenticated else {
            PendingTripInviteStorage.shared.setPendingToken(token)
            return
        }
        await joinTripAndNavigate(token: token)
    }

    func processPendingInviteIfNeeded() async {
        guard TokenStorage.shared.isAuthenticated else { return }
        guard let token = PendingTripInviteStorage.shared.consumePendingToken() else { return }
        await joinTripAndNavigate(token: token)
    }

    private func joinTripAndNavigate(token: String) async {
        do {
            let join = try await networkService.joinTripByToken(token: token)
            let response = try await networkService.getTrip(id: join.tripId)
            let trip = await loadTripForNavigation(from: response)
            SelectedTripStorage.shared.selectTrip(id: trip.id)
            router.popToRoot()
            router.navigateToTripInfo(trip: trip, onTripUpdated: nil)
        } catch {
            showToast(PinzBaseStrings.TripMembers.Invite.deepLinkJoinFailed)
        }
    }
}

private extension TripInviteDeepLinkCoordinator {

    func loadTripForNavigation(from response: GetTripResponseDTO) async -> Trip {
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
