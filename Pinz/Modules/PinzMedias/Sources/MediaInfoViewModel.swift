import PinzDomain
import PinzNetworking
import PinzBase
import PinzUI
import Observation

@MainActor
@Observable
public class MediaInfoViewModel {

    public enum Intent {
        case updatePrivacy(PrivacyIcon)
        case deleteMediaFromPin
    }

    var media: MediaItem
    private let networkService: any NetworkServiceProtocol
    private let updateAction: MediaUpdateAction?
    private let pinIdForServerMediaDelete: String?
    private let pinResponseAction: PinResponseAction?
    private let allowsMediaPrivacyChange: Bool

    private var showToast: ((String) -> Void)?
    private var router: AppRouting?

    private(set) var isDeletingPinMedia = false

    public init(
        media: MediaItem,
        updateAction: MediaUpdateAction? = nil,
        pinIdForServerMediaDelete: String? = nil,
        pinResponseAction: PinResponseAction? = nil,
        allowsMediaPrivacyChange: Bool = true,
        networkService: any NetworkServiceProtocol = NetworkService.shared
    ) {
        self.media = media
        self.updateAction = updateAction
        self.pinIdForServerMediaDelete = pinIdForServerMediaDelete
        self.pinResponseAction = pinResponseAction
        self.allowsMediaPrivacyChange = allowsMediaPrivacyChange
        self.networkService = networkService
    }

    public func setShowToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    var initialPrivacySelection: PrivacyIcon {
        PrivacyIcon.from(isPrivate: media.isPrivate)
    }

    var canDeleteMediaFromPin: Bool {
        guard let raw = pinIdForServerMediaDelete else { return false }
        let pinId = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !pinId.isEmpty else { return false }
        guard let tripId = media.tripId, !tripId.isEmpty,
              let mediaId = media.mediaId, !mediaId.isEmpty
        else { return false }
        return true
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .updatePrivacy(selection):
            guard allowsMediaPrivacyChange else { return }
            Task { @MainActor [weak self] in
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
        case .deleteMediaFromPin:
            Task { @MainActor [weak self] in
                await self?.performDeletePinMedia()
            }
        }
    }

    private func performDeletePinMedia() async {
        guard !isDeletingPinMedia else { return }
        guard let tripId = media.tripId,
              let rawPinId = pinIdForServerMediaDelete
        else { return }
        let pinId = rawPinId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !pinId.isEmpty, let mediaId = media.mediaId, !mediaId.isEmpty else { return }

        isDeletingPinMedia = true
        defer { isDeletingPinMedia = false }

        do {
            let response = try await networkService.deletePinMedia(tripId: tripId, pinId: pinId, mediaId: mediaId)
            pinResponseAction?.action(response)
            showToast?(PinzBaseStrings.MediaInfo.Toast.mediaDeleted)
            router?.pop(by: 1)
        } catch {
            if (error as? HTTPError) == .preconditionFailed {
                showToast?(PinzBaseStrings.MediaInfo.Toast.mediaDeletePinMustHaveMedia)
            } else {
                showToast?(PinzBaseStrings.MediaInfo.Toast.mediaDeleteFailed)
            }
        }
    }
}
