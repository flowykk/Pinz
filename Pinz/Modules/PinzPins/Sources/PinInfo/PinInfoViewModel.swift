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
        case startAddMedia
    }

    static let tagMaxLength = 15
    static let pinTagsMaxCount = 10
    static let pinNameMaxLength = 50
    static let pinDescriptionMaxLength = 5000

    var state: State = .info
    var previousState: State = .info

    var pin: Pin
    private var editingSnapshot: Pin?
    private let updateAction: PinUpdateAction?
    private let deleteAction: PinDeleteAction?
    private let networkService: any NetworkServiceProtocol
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?

    private(set) var isSaving = false

    private(set) var isStartingAddMedia = false

    private(set) var hasActivePinUploadAdditionSession = false

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
        refreshPinUploadAdditionSessionFlag()
    }

    public func onDisappear() {
        updateAction?.action(pin)
    }

    public func applyPinAfterAdditionUpload(_ updatedPin: Pin) {
        let matchesServer =
            updatedPin.serverId != nil
            && pin.serverId != nil
            && updatedPin.serverId == pin.serverId
        let matchesIdentity = updatedPin.id == pin.id
        guard matchesServer || matchesIdentity else { return }
        pin = updatedPin
        updateAction?.action(pin)
        refreshPinUploadAdditionSessionFlag()
    }

    public func refreshPinUploadAdditionSessionFlag() {
        guard let tripId = pin.tripId,
              let pinId = pin.serverId?.trimmingCharacters(in: .whitespacesAndNewlines),
              !pinId.isEmpty
        else {
            hasActivePinUploadAdditionSession = false
            return
        }
        hasActivePinUploadAdditionSession =
            PinUploadAdditionSessionStorage.shared.sessionId(tripId: tripId, pinId: pinId) != nil
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
        case .startAddMedia:
            await startAddMediaToPin()
        }
    }

    private func startAddMediaToPin() async {
        guard state != .editing else { return }
        guard !isSaving else { return }
        guard let tripId = pin.tripId,
              let pinId = pin.serverId?.trimmingCharacters(in: .whitespacesAndNewlines),
              !pinId.isEmpty
        else { return }

        isStartingAddMedia = true
        defer {
            isStartingAddMedia = false
            refreshPinUploadAdditionSessionFlag()
        }

        await PinUploadEntryResolver.resumeAddition(
            tripId: tripId,
            pinId: pinId,
            networkService: networkService,
            router: router,
            showToast: showToast
        )
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
        case let .addTag(newTag):
            let trimmed = newTag.tag.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.isEmpty {
                showToast?(PinzBaseStrings.PinInfo.Toast.tagEmpty)
                return
            }
            if trimmed.count > Self.tagMaxLength {
                showToast?(PinzBaseStrings.PinInfo.Toast.tagTooLong(Self.tagMaxLength))
                return
            }
            if pin.tags.contains(where: { $0.tag.caseInsensitiveCompare(trimmed) == .orderedSame }) {
                showToast?(PinzBaseStrings.PinInfo.Toast.tagAlreadyExists)
                return
            }
            if pin.tags.count >= Self.pinTagsMaxCount {
                showToast?(PinzBaseStrings.PinInfo.Toast.tagLimitReached(Self.pinTagsMaxCount))
                return
            }
            pin.tags.append(MediaTag(tag: trimmed))
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
                router?.setPinUploadAdditionSuccessHandler(nil)
                router?.pop()
            }
        case let .updatePrivacy(selection):
            Task { [weak self] in
                guard let self,
                      let tripId = pin.tripId,
                      let pinId = pin.serverId else { return }
                do {
                    let response = try await networkService.setPinPrivacy(
                        tripId: tripId,
                        pinId: pinId,
                        privacyLevel: selection.apiValue
                    )
                    pin.isPrivate = response.privacyLevel.lowercased() == "private"
                    updateAction?.action(pin)
                } catch {
                    showToast?(PinzBaseStrings.PinInfo.Toast.privacyFailed)
                }
            }
        case .deletePin:
            Task { [weak self] in
                guard let self,
                      let tripId = pin.tripId,
                      let pinId = pin.serverId else { return }
                do {
                    _ = try await networkService.deletePin(tripId: tripId, pinId: pinId)
                    let deletedPin = pin
                    showToast?(PinzBaseStrings.PinInfo.Toast.pinDeleted)
                    router?.setPinUploadAdditionSuccessHandler(nil)
                    router?.pop()
                    deleteAction?.action(deletedPin)
                } catch {
                    showToast?(PinzBaseStrings.PinInfo.Toast.deleteFailed)
                }
            }
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    public func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    // MARK: - Validation

    func validateDates() -> String? {
        guard let start = pin.startDate, let end = pin.endDate, start > end else { return nil }
        return PinzBaseStrings.PinInfo.Toast.startDateAfterEndDate
    }

    private func validateForSave() -> String? {
        let trimmedName = pin.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedName.isEmpty { return PinzBaseStrings.PinInfo.Toast.nameEmpty }
        if trimmedName.count > Self.pinNameMaxLength { return PinzBaseStrings.PinInfo.Toast.nameTooLong }
        if let description = pin.description, description.count > Self.pinDescriptionMaxLength {
            return PinzBaseStrings.PinInfo.Toast.descriptionTooLong
        }
        if let dateError = validateDates() { return dateError }
        return nil
    }

    // MARK: - Save

    private func saveEdits() async throws {
        guard let snapshot = editingSnapshot else { return }

        if let validationError = validateForSave() {
            showToast?(validationError)
            return
        }

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

        do {
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
            showToast?(PinzBaseStrings.PinInfo.Toast.pinSaved)
        } catch {
            showToast?(PinzBaseStrings.PinInfo.Toast.saveFailed)
            throw error
        }
    }

    private func applyPinFromResponse(_ response: PinResponseDTO, tripId: String) {
        var updated = response.toPin(tripId: tripId, nameIfMissing: pin.name)
        updated.issues = pin.issues
        pin = updated
        updateAction?.action(pin)
    }

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
            showToast?(PinzBaseStrings.PinInfo.Toast.locationUpdated)
        } catch {
            pin.coordinates = previous
            showToast?(PinzBaseStrings.PinInfo.Toast.locationFailed)
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

    var addMediaButtonDisabled: Bool {
        state == .editing || isSaving || isStartingAddMedia || !canStartPinUploadAddition
    }

    private var canStartPinUploadAddition: Bool {
        guard let tripId = pin.tripId, !tripId.isEmpty else { return false }
        guard let pinId = pin.serverId?.trimmingCharacters(in: .whitespacesAndNewlines), !pinId.isEmpty else {
            return false
        }
        return true
    }
}
