import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class WishlistElementViewModel {

    static let placeNameMaxLength = 50
    static let placeDescriptionMaxLength = 5000

    enum State {
        case `default`
        case editing
    }

    enum Route {
        case back
    }

    enum Intent {
        case selectPhoto(PhotosPickerItem)
        case edit
        case endEdit
        case delete

        case navigate(Route)
    }

    var element: DesiredPlace
    var localImage: UIImage?
    var state: State = .default
    var isLoading = false

    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?

    init(element: DesiredPlace, networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.element = element
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else {
                    showToast?(PinzBaseStrings.WishlistElement.Toast.photoLoadFailed)
                    return
                }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        localImage = uiImage
                    }
                } else {
                    showToast?(PinzBaseStrings.WishlistElement.Toast.photoLoadFailed)
                }
            }
        case .edit:
            changeState(to: .editing)
        case .endEdit:
            if let validationError = validateForSave() {
                showToast?(validationError)
                return
            }
            let trimmedName = element.name.trimmingCharacters(in: .whitespacesAndNewlines)
            element.name = trimmedName
            changeState(to: .default)
            isLoading = true
            Task {
                defer { isLoading = false }
                var imageS3Key: String?
                if let image = localImage {
                    guard let data = image.jpegData(compressionQuality: 0.8) else {
                        showToast?(PinzBaseStrings.WishlistElement.Toast.imagePrepareFailed)
                        return
                    }
                    do {
                        let uploadResp = try await networkService.requestDesiredPlaceImageUpload(
                            filename: "place_\(UUID().uuidString).jpg",
                            contentType: "image/jpeg"
                        )
                        try await networkService.uploadToS3(url: uploadResp.uploadUrl, data: data, contentType: "image/jpeg")
                        imageS3Key = uploadResp.s3Key
                    } catch {
                        showToast?(PinzBaseStrings.WishlistElement.Toast.imageUploadFailed)
                        return
                    }
                }
                do {
                    let updated = try await networkService.updateDesiredPlace(
                        placeId: element.id,
                        name: element.name,
                        description: element.description,
                        imageS3Key: imageS3Key
                    )
                    element = updated.toDesiredPlace()
                    localImage = nil
                    showToast?(PinzBaseStrings.WishlistElement.Toast.placeUpdated)
                } catch {
                    showToast?(PinzBaseStrings.WishlistElement.Toast.updateFailed)
                }
            }
        case .delete:
            isLoading = true
            Task {
                defer { isLoading = false }
                do {
                    _ = try await networkService.deleteDesiredPlace(placeId: element.id)
                    showToast?(PinzBaseStrings.WishlistElement.Toast.placeDeleted)
                    router?.pop()
                } catch {
                    showToast?(PinzBaseStrings.WishlistElement.Toast.deleteFailed)
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

    private func validateForSave() -> String? {
        let trimmedName = element.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedName.isEmpty { return PinzBaseStrings.WishlistElement.Toast.nameEmpty }
        if !trimmedName.isValidWishlistPlaceName {
            return PinzBaseStrings.WishlistElement.Toast.nameInvalidChars
        }
        if trimmedName.count > Self.placeNameMaxLength {
            return PinzBaseStrings.WishlistElement.Toast.nameTooLong(Self.placeNameMaxLength)
        }
        if element.description.count > Self.placeDescriptionMaxLength {
            return PinzBaseStrings.WishlistElement.Toast.descriptionTooLong(Self.placeDescriptionMaxLength)
        }
        return nil
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
