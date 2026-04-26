import SwiftUI
import PhotosUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class WishlistElementViewModel {

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

    init(element: DesiredPlace, networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.element = element
        self.networkService = networkService
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else { return }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        localImage = uiImage
                    }
                }
            }
        case .edit:
            changeState(to: .editing)
        case .endEdit:
            changeState(to: .default)
            isLoading = true
            Task {
                defer { isLoading = false }
                do {
                    var imageS3Key: String?
                    if let image = localImage {
                        let uploadResp = try await networkService.requestDesiredPlaceImageUpload(
                            filename: "place_\(UUID().uuidString).jpg",
                            contentType: "image/jpeg"
                        )
                        if let data = image.jpegData(compressionQuality: 0.8) {
                            try await networkService.uploadToS3(url: uploadResp.uploadUrl, data: data, contentType: "image/jpeg")
                            imageS3Key = uploadResp.s3Key
                        }
                    }
                    let updated = try await networkService.updateDesiredPlace(
                        placeId: element.id,
                        name: element.name,
                        description: element.description,
                        imageS3Key: imageS3Key
                    )
                    element = updated.toDesiredPlace()
                    localImage = nil
                } catch {}
            }
        case .delete:
            isLoading = true
            Task {
                defer { isLoading = false }
                do {
                    _ = try await networkService.deleteDesiredPlace(placeId: element.id)
                    router?.pop()
                } catch {}
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

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
