import PinzDomain
import PinzNetworking
import PinzUI
import Observation

@Observable
public class MediaInfoViewModel {

    public enum Intent {
        case updatePrivacy(PrivacyIcon)
    }

    var media: MediaItem
    private let networkService: any NetworkServiceProtocol
    private let updateAction: MediaUpdateAction?

    public init(media: MediaItem, updateAction: MediaUpdateAction? = nil, networkService: any NetworkServiceProtocol = NetworkService.shared) {
        self.media = media
        self.updateAction = updateAction
        self.networkService = networkService
    }

    var initialPrivacySelection: PrivacyIcon {
        PrivacyIcon.from(isPrivate: media.isPrivate)
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .updatePrivacy(selection):
            Task { [weak self] in
                guard let self,
                      let tripId = media.tripId,
                      let mediaId = media.mediaId else { return }
                guard let response = try? await networkService.setMediaPrivacy(
                    tripId: tripId,
                    mediaId: mediaId,
                    privacyLevel: selection.apiValue
                ) else { return }
                media.isPrivate = response.privacyLevel.lowercased() == "private"
                updateAction?.action(media)
            }
        }
    }
}
