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

        case storageSettings
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
        case deleteAvatar
        case navigate(Route)
    }

    var state: State = .default
    var isLoading = false

    var user: User
    var userImage: UIImage?
    private var showToast: ((String) -> Void)?
    private let networkService: NetworkServiceProtocol
    private var router: AppRouting?
    private var avatarUploadTask: Task<ProfileResponseDTO, Error>?

    private enum AvatarUploadFlowError: Error {
        case missingImageData
        case invalidUploadResponse
    }

    public init(user: User, networkService: NetworkServiceProtocol = NetworkService.shared) {
        self.user = user
        self.networkService = networkService
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
                uploadAvatarTask(with: newImage)
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
            guard !isLoading else { return }

            let trimmed = user.nickname.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else {
                showToast?(PinzBaseStrings.Profile.Toast.nicknameEmpty)
                return
            }
            guard trimmed.count >= 4, trimmed.count <= 20 else {
                showToast?(PinzBaseStrings.Profile.Toast.nicknameLengthInvalid)
                return
            }
            let allowedChars = CharacterSet(charactersIn: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-абвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ")
            guard trimmed.unicodeScalars.allSatisfy({ allowedChars.contains($0) }) else {
                showToast?(PinzBaseStrings.Profile.Toast.nicknameInvalidChars)
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
        case .deleteAvatar:
            guard !isLoading else {
                return
            }

            isLoading = true

            Task {
                await deleteAvatar()
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
            case .storageSettings:
                router?.navigateToStorageSettings()
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

    public func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
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
            showToast?(PinzBaseStrings.Profile.Toast.loadFailed)
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
            if let uploadTask = avatarUploadTask {
                do {
                    _ = try await uploadTask.value
                } catch is CancellationError {
                    print("[Profile] Avatar upload canceled")
                } catch let error as MediaUploadError {
                    switch error {
                    case .limitExceeded(let kind, _, _) where kind == .image:
                        showToast?(MediaUploadPreprocessor.localizedLimitMessage(for: kind))
                        return
                    default:
                        showToast?(PinzBaseStrings.Profile.Toast.avatarUploadFailed)
                        return
                    }
                } catch {
                    print("[Profile] Failed to upload avatar before save: \(error)")
                    showToast?(PinzBaseStrings.Profile.Toast.avatarUploadFailed)
                    return
                }
                avatarUploadTask = nil
            }

            let trimmed = user.nickname.trimmingCharacters(in: .whitespacesAndNewlines)
            let response = try await networkService.updateProfile(username: trimmed)
            user = response.toUser()
            userImage = nil
            router?.notifyCurrentProfileUpdated(user)
            showToast?(PinzBaseStrings.Profile.Toast.profileSaved)
        } catch {
            print("[Profile] Failed to update profile: \(error)")
            showToast?(PinzBaseStrings.Profile.Toast.saveFailed)
        }
    }

    private func uploadAvatarTask(with image: UIImage) {
        avatarUploadTask?.cancel()

        avatarUploadTask = Task { @MainActor [weak self] in
            guard let self else {
                throw CancellationError()
            }
            defer {
                self.avatarUploadTask = nil
            }

            let response = try await self.uploadAvatarFlow(image: image)

            self.user = response.toUser()
            self.userImage = nil
            self.router?.notifyCurrentProfileUpdated(self.user)

            return response
        }
    }

    private func uploadAvatarFlow(image: UIImage) async throws -> ProfileResponseDTO {
        let contentType: String

        if let jpegData = image.jpegData(compressionQuality: 0.85) {
            contentType = "image/jpeg"
        } else if let pngData = image.pngData() {
            contentType = "image/png"
        } else {
            throw AvatarUploadFlowError.missingImageData
        }

        let filename = "avatar-\(UUID().uuidString).\(contentType == "image/png" ? "png" : "jpg")"
        let request = try await networkService.requestAvatarUpload(filename: filename, contentType: contentType)

        guard let uploadUrl = request.uploadUrl,
              let s3Key = request.s3Key,
              !uploadUrl.isEmpty,
              !s3Key.isEmpty else {
            throw AvatarUploadFlowError.invalidUploadResponse
        }

        let prepared = try await MediaUploadPreprocessor.shared.prepareImage(
            image,
            contentType: contentType,
            uploadURL: uploadUrl,
            context: "avatar"
        )
        switch prepared.body {
        case let .data(data):
            try await networkService.uploadToS3(url: uploadUrl, data: data, contentType: prepared.contentType)
        case .file:
            throw AvatarUploadFlowError.invalidUploadResponse
        }
        return try await networkService.confirmAvatarUpload(s3Key: s3Key)
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
            showToast?(PinzBaseStrings.Profile.Toast.accountDeleteFailed)
        }
    }

    private func deleteAvatar() async {
        defer {
            withAnimation(.easeInOut(duration: 0.3)) {
                isLoading = false
            }
        }

        avatarUploadTask?.cancel()
        avatarUploadTask = nil
        userImage = nil

        do {
            let response = try await networkService.deleteAvatar()
            user = response.toUser()
            userImage = nil
            router?.notifyCurrentProfileUpdated(user)
        } catch {
            print("[Profile] Failed to delete avatar: \(error)")
            showToast?(PinzBaseStrings.Profile.Toast.avatarDeleteFailed)
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
