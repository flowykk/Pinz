import SwiftUI
import PinzNetworking
import PinzDomain
import PinzUI
import PinzBase

@Observable
public class PinInfoViewModel {
    
    public enum State: SegmentedItem {
        public var id: Self { self }

        case info
        case gallery
        case editing

        public var content: SegmentedItemContent {
            switch self {
            case .info:
                .text(PinzBaseStrings.Common.Label.info)
            case .gallery:
                .text(PinzBaseStrings.Common.Label.gallery)
            case .editing:
                .text("")
            }
        }
    }

    public enum Route {
        case mediaInfo(MediaItem)
        case changePlace
        case back
    }

    public enum Intent {
        case edit
        case endEdit

        case addTag(MediaTag)
        case deleteTag(MediaTag)

        case navigate(Route)
        case updatePrivacy(PrivacyIcon)
    }

    var state: State = .info
    var previousState: State = .info

    var pin: Pin
    private let updateAction: PinUpdateAction?
    private let networkService = NetworkService.shared
    private var router: AppRouting?

    var isEditing: Bool {
        state == .editing
    }

    public init(pin: Pin, updateAction: PinUpdateAction? = nil) {
        self.pin = pin
        self.updateAction = updateAction
    }

    public func onDisappear() {
        updateAction?.action(pin)
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .edit:
            previousState = state
            changeState(to: .editing)
        case .endEdit:
            changeState(to: previousState)
        case let .addTag(tag):
            pin.tags.append(tag)
        case let .deleteTag(tag):
            pin.tags.removeAll { $0.tag == tag.tag }
        case let .navigate(route):
            switch route {
            case let .mediaInfo(media):
                router?.navigateToMediaInfo(media: media, updateAction: MediaUpdateAction { [weak self] updatedMedia in
                    guard let self, let idx = pin.medias.firstIndex(where: { $0.mediaId == updatedMedia.mediaId }) else { return }
                    pin.medias[idx] = updatedMedia
                })
            case .changePlace:
                let action = PlaceSaveAction { [weak self] coordinate in
                    self?.pin.coordinates = coordinate
                }
                router?.navigateToPinPlaceChange(pin: pin, action: action)
            case .back:
                router?.pop()
            }
        case let .updatePrivacy(selection):
            Task { [weak self] in
                guard let self,
                      let tripId = pin.tripId,
                      let pinId = pin.serverId else { return }
                guard let response = try? await networkService.setPinPrivacy(tripId: tripId, pinId: pinId, privacyLevel: selection.apiValue) else { return }
                pin.isPrivate = response.privacyLevel.lowercased() != "public"
                updateAction?.action(pin)
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
