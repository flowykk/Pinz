import SwiftUI
import PinzNetworking
import PinzBase

struct TripMember: Equatable, Identifiable {
    let id: String
    let username: String
    let avatar: UIImage
}

@MainActor @Observable
final class TripMembersViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    var isLoading: Bool = false
    var searchText: String = ""

    var filteredMembers: [TripMember] {
        guard !searchText.isEmpty else { return members }
        return members.filter { $0.username.localizedCaseInsensitiveContains(searchText) }
    }

    var members: [TripMember] = [
        TripMember(id: "1", username: "alex_travel", avatar: UIImage(systemName: "person.circle.fill")!),
        TripMember(id: "2", username: "maria_k", avatar: UIImage(systemName: "person.circle.fill")!),
        TripMember(id: "3", username: "den_explore", avatar: UIImage(systemName: "person.circle.fill")!),
        TripMember(id: "4", username: "julia_photo", avatar: UIImage(systemName: "person.circle.fill")!),
    ]

    private let networkService = NetworkService.shared
    private var router: AppRouting?

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
