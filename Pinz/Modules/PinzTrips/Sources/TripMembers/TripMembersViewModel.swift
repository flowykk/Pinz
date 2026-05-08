import SwiftUI
import PinzNetworking
import PinzBase
import PinzDomain

enum TripMemberRole: String {
    case admin
    case member
}

struct TripMember: Equatable, Identifiable {
    let id: String
    let username: String
    let avatar: UIImage
    let role: TripMemberRole?
}

@MainActor @Observable
final class TripMembersViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case openProfile(TripMember)
    }

    var isLoading: Bool = false
    var searchText: String = ""

    var filteredMembers: [TripMember] {
        guard !searchText.isEmpty else { return members }
        return members.filter { $0.username.localizedCaseInsensitiveContains(searchText) }
    }

    var members: [TripMember] = []
    let currentUserId: String?

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    init(participants: [TripParticipantDTO], currentUserId: String?) {
        self.currentUserId = currentUserId
        let placeholder = ImageProviderType.user.placeholder
        members = participants.map {
            TripMember(id: $0.userId, username: $0.username, avatar: placeholder, role: TripMemberRole(rawValue: $0.role ?? ""))
        }
        Task { await loadAvatars(for: participants) }
    }

    private func loadAvatars(for participants: [TripParticipantDTO]) async {
        for participant in participants {
            guard let url = participant.avatarUrl else { continue }
            guard let image = await ImageProvider.loadOrGetImage(
                for: url,
                .user,
                cacheVariant: .thumbnail,
                targetPixel: 120
            ) else { continue }
            guard let idx = members.firstIndex(where: { $0.id == participant.userId }) else { continue }
            members[idx] = TripMember(id: participant.userId, username: participant.username, avatar: image, role: TripMemberRole(rawValue: participant.role ?? ""))
        }
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .openProfile(member):
            router?.navigateToPublicProfile(userId: member.id)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
