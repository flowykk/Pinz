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

    var isCompleteButtonDisabled: Bool {
        switch state {
        case .name:
            return name.isEmpty
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
            case .name:        changeState(to: .description)
            case .description: changeState(to: .photo)
            case .photo:
                guard let image else { return }
                isLoading = true
                Task {
                    defer { isLoading = false }
                    do {
                        let uploadResp = try await networkService.requestDesiredPlaceImageUpload(
                            filename: "place_\(UUID().uuidString).jpg",
                            contentType: "image/jpeg"
                        )
                        guard let data = image.jpegData(compressionQuality: 0.8) else { return }
                        try await networkService.uploadToS3(url: uploadResp.uploadUrl, data: data, contentType: "image/jpeg")
                        let dto = try await networkService.createDesiredPlace(
                            name: name, description: description, s3Key: uploadResp.s3Key
                        )
                        onCreated(dto.toDesiredPlace())
                        router?.pop()
                    } catch {}
                }
            }
        case let .selectPhoto(item):
            Task {
                guard let loaded = await MediaLoader.shared.load(from: item) else { return }
                if case let .image(uiImage) = loaded.content {
                    withAnimation(.easeInOut(duration: 0.3)) {
                        image = uiImage
                    }
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

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}
