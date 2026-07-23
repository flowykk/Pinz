import Foundation
import PinzBase
import PinzNetworking

@MainActor @Observable
final class TripInviteViewModel {

    var isLoading = true
    var inviteUrl: String?
    var errorMessage: String?

    private let tripId: String
    private let networkService: NetworkServiceProtocol

    init(tripId: String, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.tripId = tripId
        self.networkService = networkService
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        inviteUrl = nil
        do {
            let response = try await networkService.generateInviteLink(tripId: tripId, expiresInSeconds: nil)
            inviteUrl = response.effectiveInviteUrl
        } catch {
            errorMessage = PinzBaseStrings.TripMembers.Invite.error
        }
        isLoading = false
    }
}
