import SwiftUI
import PinzNetworking
import PinzDomain
import PinzUI
import PinzBase

@MainActor
@Observable
public class ProfileViewModel {

    public enum State {
        case `default`
        case editing
    }

    public enum Route {
        case emailChange

        case statistics

        case trips
        case wishlist
        case saved

        case notifications
        case appearance

        case back
    }

    public enum Intent {
        case changeState
        case setImage(UIImage?)
        case getProfile
        case saveProfile
        case deleteAccount
        case navigate(Route)
    }

    var state: State = .default
    var isLoading = false

    var user: User
    var userImage: UIImage?
    private let networkService = NetworkService.shared
    private var router: AppRouting?

    public init(user: User) {
        self.user = user
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .changeState:
            switch state {
            case .default: changeState(to: .editing)
            case .editing: changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                userImage = newImage
            }
        case .getProfile:
            guard !isLoading else {
                return
            }

            isLoading = true

            Task {
                await loadProfile()
            }
        case .saveProfile:
            guard !isLoading else {
                return
            }

            isLoading = true

            Task {
                await saveProfile()
            }
        case .deleteAccount:
            guard !isLoading else {
                return
            }

            isLoading = true

            Task {
                await deleteAccount()
            }
        case let .navigate(route):
            switch route {
            case .emailChange:
                let action = EmailChangeAction { [weak self] newEmail in
                    self?.user.email = newEmail
                    self?.router?.pop()
                }
                router?.navigateToEmailChange(email: user.email, userId: user.profileId, action: action)
            case .statistics:
                router?.navigateToStatistics()
            case .trips:
                router?.navigateToTrips()
            case .wishlist:
                router?.navigateToPlacesWishlist()
            case .saved:
                router?.navigateToSavedMaps()
            case .notifications:
                router?.navigateToNotifications()
            case .appearance:
                router?.navigateToAppearance()
            case .back:
                router?.pop()
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func loadProfile() async {
        withAnimation(.easeInOut(duration: 0.3)) {
            isLoading = true
        }
        defer {
            withAnimation(.easeInOut(duration: 0.3)) {
                isLoading = false
            }
        }

        do {
            let response = try await networkService.getProfile()
            user = response.toUser()
            userImage = nil
        } catch {
            print("[Profile] Failed to get profile: \(error)")
        }
    }

    private func saveProfile() async {
        withAnimation(.easeInOut(duration: 0.3)) {
            isLoading = true
        }
        defer {
            withAnimation(.easeInOut(duration: 0.3)) {
                isLoading = false
            }
            changeState(to: .default)
        }

        do {
            let trimmed = user.nickname.trimmingCharacters(in: .whitespacesAndNewlines)
            let response = try await networkService.updateProfile(username: trimmed)
            user = response.toUser()
        } catch {
            print("[Profile] Failed to update profile: \(error)")
        }
    }

    private func deleteAccount() async {
        defer {
            withAnimation(.easeInOut(duration: 0.3)) {
                isLoading = false
            }
        }

        do {
            _ = try await networkService.deleteAccount()
            router?.navigateToMain()
        } catch {
            print("[Profile] Failed to delete account: \(error)")
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
