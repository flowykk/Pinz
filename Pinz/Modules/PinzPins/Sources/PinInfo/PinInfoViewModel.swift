import CoreLocation
import SwiftUI
import PinzNetworking
import PinzDomain
import PinzBase
import PinzUI

@MainActor
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
        case cancelEdit

        case addTag(MediaTag)
        case deleteTag(MediaTag)

        case navigate(Route)
        case updatePrivacy(PrivacyIcon)
        case deletePin
    }

    public enum AsyncIntent {
        case saveEdits
    }

    var state: State = .info
    var previousState: State = .info

    var pin: Pin
    private var editingSnapshot: Pin?
    private let updateAction: PinUpdateAction?
    private let deleteAction: PinDeleteAction?
    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?

    /// True while a save request is in flight (disables Done, like Profile `isLoading` on save).
    private(set) var isSaving = false

    public init(
        pin: Pin,
        updateAction: PinUpdateAction? = nil,
        deleteAction: PinDeleteAction? = nil,
        networkService: any NetworkServiceProtocol = NetworkService.shared
    ) {
        self.pin = pin
        self.updateAction = updateAction
        self.deleteAction = deleteAction
        self.networkService = networkService
    }

    public func onDisappear() {
        updateAction?.action(pin)
    }

    public func asyncDispatch(
        _ intent: AsyncIntent,
        onError: ((Error) -> Void)? = nil
    ) async {
        do {
            try await executeAsyncIntent(intent)
        } catch {
            onError?(error)
        }
    }

    private func executeAsyncIntent(_ intent: AsyncIntent) async throws {
        switch intent {
        case .saveEdits:
            try await saveEdits()
        }
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .edit:
            backupPinForEditing()
            previousState = state
            changeState(to: .editing)
        case .cancelEdit:
            restorePinFromSnapshot()
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
                    self?.applyPlaceChange(coordinate)
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
                guard let response = try? await networkService.setPinPrivacy(
                    tripId: tripId,
                    pinId: pinId,
                    privacyLevel: selection.apiValue
                ) else { return }
                pin.isPrivate = response.privacyLevel.lowercased() == "private"
                updateAction?.action(pin)
            }
        case .deletePin:
            Task { [weak self] in
                guard let self,
                      let tripId = pin.tripId,
                      let pinId = pin.serverId else { return }
                _ = try? await networkService.deletePin(tripId: tripId, pinId: pinId)
                let deletedPin = pin
                router?.pop()
                deleteAction?.action(deletedPin)
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    // MARK: - Save

    private func saveEdits() async throws {
        guard let snapshot = editingSnapshot else { return }
        let tripId = pin.tripId
        let pinId = pin.serverId

        if tripId == nil || pinId == nil {
            // Local / stub: exit edit without API.
            applyExitFromEditAfterSuccessKeepingCurrentPin()
            return
        }

        let diff = makePatch(from: snapshot, to: pin)
        if !diff.hasAnyFieldToSend {
            applyExitFromEditAfterSuccessKeepingCurrentPin()
            return
        }

        isSaving = true
        defer { isSaving = false }

        let response = try await networkService.updatePin(
            tripId: tripId!,
            pinId: pinId!,
            name: diff.name,
            description: diff.description,
            category: diff.category,
            latitude: diff.latitude,
            longitude: diff.longitude,
            startTimeUnix: diff.startTimeUnix,
            endTimeUnix: diff.endTimeUnix,
            tags: diff.tags,
            tagsSet: diff.tagsSet
        )

        applyPinFromResponse(response, tripId: tripId!)
        applyExitFromEditAfterSuccessKeepingCurrentPin()
    }

    private func applyPinFromResponse(_ response: PinResponseDTO, tripId: String) {
        var updated = response.pin.toPin(
            index: 0,
            tripId: tripId,
            nameIfMissing: pin.name
        )
        updated.issues = pin.issues
        pin = updated
        updateAction?.action(pin)
    }

    /// After PinPlaceChange: optimistically set coordinates, then PATCH; revert coordinates on failure.
    private func applyPlaceChange(_ newCoordinate: CLLocationCoordinate2D?) {
        let previous = pin.coordinates
        if coordinatesEqual(previous, newCoordinate) { return }
        pin.coordinates = newCoordinate
        guard let tripId = pin.tripId, let pinId = pin.serverId, let c = newCoordinate else { return }
        Task { @MainActor [weak self] in
            await self?.patchLocation(
                tripId: tripId,
                pinId: pinId,
                coordinate: c,
                revertTo: previous
            )
        }
    }

    private func patchLocation(
        tripId: String,
        pinId: String,
        coordinate: CLLocationCoordinate2D,
        revertTo previous: CLLocationCoordinate2D?
    ) async {
        do {
            let response = try await networkService.updatePin(
                tripId: tripId,
                pinId: pinId,
                name: nil,
                description: nil,
                category: nil,
                latitude: coordinate.latitude,
                longitude: coordinate.longitude,
                startTimeUnix: nil,
                endTimeUnix: nil,
                tags: nil,
                tagsSet: nil
            )
            applyPinFromResponse(response, tripId: tripId)
        } catch {
            pin.coordinates = previous
            #if DEBUG
            print("[PinInfo] PATCH location failed: \(error)")
            #endif
        }
    }

    private func applyExitFromEditAfterSuccessKeepingCurrentPin() {
        editingSnapshot = nil
        changeState(to: previousState)
    }

    private struct PinEditPatch {
        var name: String?
        var description: String?
        var category: String?
        var latitude: Double?
        var longitude: Double?
        var startTimeUnix: Int?
        var endTimeUnix: Int?
        var tags: [String]?
        var tagsSet: Bool?

        var hasAnyFieldToSend: Bool {
            name != nil
                || description != nil
                || category != nil
                || latitude != nil
                || longitude != nil
                || startTimeUnix != nil
                || endTimeUnix != nil
                || tags != nil
        }
    }

    private func makePatch(from snapshot: Pin, to current: Pin) -> PinEditPatch {
        var patch = PinEditPatch(
            name: nil,
            description: nil,
            category: nil,
            latitude: nil,
            longitude: nil,
            startTimeUnix: nil,
            endTimeUnix: nil,
            tags: nil,
            tagsSet: nil
        )

        if snapshot.name != current.name {
            patch.name = current.name
        }

        let sDesc = snapshot.description ?? ""
        let cDesc = current.description ?? ""
        if sDesc != cDesc {
            patch.description = cDesc
        }

        if snapshot.category != current.category {
            patch.category = current.category.apiValue
        }

        if !coordinatesEqual(snapshot.coordinates, current.coordinates) {
            if let c = current.coordinates {
                patch.latitude = c.latitude
                patch.longitude = c.longitude
            }
        }

        if snapshot.startDate != current.startDate {
            patch.startTimeUnix = current.startDate.map { Int($0.timeIntervalSince1970) }
        }
        if snapshot.endDate != current.endDate {
            patch.endTimeUnix = current.endDate.map { Int($0.timeIntervalSince1970) }
        }

        let oldTags = Set(snapshot.tags.map(\.tag))
        let newTags = Set(current.tags.map(\.tag))
        if oldTags != newTags {
            patch.tags = current.tags.map(\.tag)
            patch.tagsSet = true
        }

        return patch
    }

    private func coordinatesEqual(
        _ a: CLLocationCoordinate2D?,
        _ b: CLLocationCoordinate2D?
    ) -> Bool {
        switch (a, b) {
        case (nil, nil):
            return true
        case let (x?, y?):
            return x.latitude == y.latitude && x.longitude == y.longitude
        default:
            return false
        }
    }

    private func backupPinForEditing() {
        editingSnapshot = pin
    }

    private func restorePinFromSnapshot() {
        if let snapshot = editingSnapshot {
            pin = snapshot
        }
        editingSnapshot = nil
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }
}

extension PinInfoViewModel {
    var isEditing: Bool {
        state == .editing
    }
}
