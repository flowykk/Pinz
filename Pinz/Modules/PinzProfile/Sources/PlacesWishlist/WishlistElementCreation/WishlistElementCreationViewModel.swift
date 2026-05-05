import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class WishlistElementCreationViewModel {

    enum State {
        case name
        case description
        case photo
    }

    enum Route {
        case back
    }

    enum Intent {
        case `continue`
        case selectPhoto(PhotosPickerItem)
        case navigate(Route)
    }

    var state: State = .name
    var image: UIImage?
    var name: String = ""
    var description: String = ""
    var isLoading = false

    private let onCreated: (DesiredPlace) -> Void
    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?

    var isCompleteButtonDisabled: Bool {
        switch state {
        case .name:
            return name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        case .description:
            return description.isEmpty
        case .photo:
            return image == nil
        }
    }

    init(onCreated: @escaping (DesiredPlace) -> Void, networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.onCreated = onCreated
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .continue:
            switch state {
            case .name:
                guard validateAndNormalizePlaceName() else { return }
                changeState(to: .description)
            case .description:
                guard validateAndNormalizePlaceName() else { return }
                changeState(to: .photo)
            case .photo:
                guard validateAndNormalizePlaceName() else { return }
                guard let image else { return }
                isLoading = true
                Task {
                    defer { isLoading = false }
                    guard let data = image.jpegData(compressionQuality: 0.8) else {
                        showToast?(PinzBaseStrings.Wishlist.Toast.imagePrepareFailed)
                        return
                    }
                    do {
                        let uploadResp = try await networkService.requestDesiredPlaceImageUpload(
                            filename: "place_\(UUID().uuidString).jpg",
                            contentType: "image/jpeg"
                        )
                        try await networkService.uploadToS3(url: uploadResp.uploadUrl, data: data, contentType: "image/jpeg")
                        do {
                            let dto = try await networkService.createDesiredPlace(
                                name: name, description: description, s3Key: uploadResp.s3Key
                            )
                            showToast?(PinzBaseStrings.Wishlist.Toast.placeCreated)
                            onCreated(dto.toDesiredPlace())
                            router?.pop()
                        } catch {
                            showToast?(PinzBaseStrings.Wishlist.Toast.createFailed)
                        }
                    } catch {
                        showToast?(PinzBaseStrings.Wishlist.Toast.imageUploadFailed)
                    }
                }
            }
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else {
                    showToast?(PinzBaseStrings.Wishlist.Toast.photoLoadFailed)
                    return
                }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        image = uiImage
                    }
                } else {
                    showToast?(PinzBaseStrings.Wishlist.Toast.photoLoadFailed)
                }
            }
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

    public func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }

    /// Trims `name`, shows a toast and returns `false` if place-name rules are violated.
    private func validateAndNormalizePlaceName() -> Bool {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            showToast?(PinzBaseStrings.WishlistElement.Toast.nameEmpty)
            return false
        }
        guard trimmed.isValidWishlistPlaceName else {
            showToast?(PinzBaseStrings.Wishlist.Toast.nameInvalidChars)
            return false
        }
        name = trimmed
        return true
    }
}
