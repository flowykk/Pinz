import SwiftUI
import Foundation
import PinzDomain
import PinzBase
import PinzNetworking
import PinzUI

@MainActor
@Observable
final class TripInfoViewModel {

    enum State {
        case `default`
        case editing
    }

    enum Route {
        case pinsList
        case selectPins
        case back
    }

    enum Intent {
        case changeState
        case setImage(UIImage?)
        case navigate(Route)
        case updatePrivacy(PrivacyIcon)
    }

    enum AsyncIntent {
        case editTrip
        case updateNotifications(enabled: Bool)
        case leaveTrip
    }

    var state: State = .default

    var trip: Trip
    private var editingSnapshot: Trip?
    private var router: AppRouting?
    private let networkService: any NetworkServiceProtocol
    private let onTripUpdated: (() -> Void)?
    private var tripCoverUploadTask: Task<TripDTO, Error>?
    private var showToast: ((String) -> Void)?

    static let requiredBattleMediaCount = PhotoBattleViewModel.requiredBattleMediaCount
    var isPhotoBattlePresented = false
    var isStartingBattle = false
    var photoBattleViewModel: PhotoBattleViewModel?

    private enum TripCoverUploadFlowError: Error {
        case missingImageData
        case invalidUploadResponse
    }

    init(trip: Trip, networkService: NetworkServiceProtocol = NetworkService.shared, onTripUpdated: (() -> Void)? = nil) {
        self.trip = trip
        self.networkService = networkService
        self.onTripUpdated = onTripUpdated
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .changeState:
            switch state {
            case .default:
                backupTripForEditing()
                changeState(to: .editing)
            case .editing:
                restoreTripFromSnapshot()
                changeState(to: .default)
            }
        case let .setImage(newImage):
            if let newImage {
                trip.image = newImage
                uploadTripCoverTask(with: newImage)
            }
        case let .navigate(route):
            switch route {
            case .pinsList:
                router?.navigateToPinsList(trip: trip)
            case .selectPins:
                router?.navigateToSelectablePinsList(trip: trip)
            case .back:
                router?.pop()
            }
        case let .updatePrivacy(selection):
            let tripId = trip.id
            Task { [weak self] in
                guard let self else { return }
                do {
                    let response = try await networkService.setTripPrivacy(tripId: tripId, privacyLevel: selection.apiValue)
                    trip.privacyLevel = response.privacyLevel
                    onTripUpdated?()
                } catch {
                    showToast?(PinzBaseStrings.TripInfo.Toast.privacyFailed)
                }
            }
        }
    }

    func asyncDispatch(
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
        case .editTrip:
            try await editTrip()
        case let .updateNotifications(enabled):
            try await updateNotifications(enabled)
        case .leaveTrip:
            try await leaveCurrentTrip()
        }
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    func setToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    func validateDates() -> String? {
        guard let start = trip.startDate, let end = trip.endDate, start > end else { return nil }
        return PinzBaseStrings.TripInfo.Toast.startDateAfterEndDate
    }

    func deleteTrip() async {
        do {
            try await networkService.deleteTrip(id: trip.id)
            SelectedTripStorage.shared.clearSelection()
            showToast?(PinzBaseStrings.TripInfo.Toast.tripDeleted)
            dispatch(.navigate(.back))
        } catch {
            showToast?(PinzBaseStrings.TripInfo.Toast.deleteFailed)
        }
    }

    var tripMediaCount: Int {
        trip.pins.reduce(0) { $0 + $1.medias.count }
    }

    var canStartPhotoBattle: Bool {
        tripMediaCount >= Self.requiredBattleMediaCount
    }

    var photoBattleAvailabilityMessage: String? {
        canStartPhotoBattle ? nil : PinzBaseStrings.TripInfo.Message.photoBattleNeedMedia(Self.requiredBattleMediaCount)
    }

    func startPhotoBattle() async {
        guard !isStartingBattle else {
            return
        }

        guard canStartPhotoBattle else {
            showToast?(photoBattleAvailabilityMessage ?? "")
            return
        }

        isStartingBattle = true
        closePhotoBattle()

        defer {
            isStartingBattle = false
        }

        do {
            let response = try await networkService.startBattle(tripId: trip.id)
            let parsedMedia = response.media.compactMap { Self.mapToBattleMedia(from: $0) }
            let battleMedia = Array(parsedMedia.prefix(Self.requiredBattleMediaCount))

            guard battleMedia.count == Self.requiredBattleMediaCount else {
                showToast?(PinzBaseStrings.TripInfo.Message.photoBattleStartFailed)
                return
            }

            let battleSessionId = response.battleId.isEmpty ? UUID().uuidString : response.battleId
            photoBattleViewModel = PhotoBattleViewModel(
                tripId: trip.id,
                battleSessionId: battleSessionId,
                media: battleMedia,
                networkService: networkService,
                onFinish: { [weak self] in
                    self?.closePhotoBattle()
                }
            )
            isPhotoBattlePresented = true
            Task {
                await photoBattleViewModel?.preloadBattleMedia()
            }
        } catch let error as HTTPError where error == .preconditionFailed {
            showToast?(PinzBaseStrings.TripInfo.Message.photoBattleNeedMediaWithContext(Self.requiredBattleMediaCount))
        } catch {
            showToast?(PinzBaseStrings.TripInfo.Message.photoBattleStartFailedGeneric)
        }
    }

    func dismissPhotoBattle() {
        closePhotoBattle()
    }

    private func closePhotoBattle() {
        isPhotoBattlePresented = false
        photoBattleViewModel = nil
    }

    private static func mapToBattleMedia(from dto: StartBattleMediaDTO) -> PhotoBattleMedia? {
        guard !dto.photoBattleMediaId.isEmpty else { return nil }
        guard let mediaURL = URL(string: dto.url) else { return nil }
        return PhotoBattleMedia(
            photoBattleMediaId: dto.photoBattleMediaId,
            url: mediaURL,
            kind: dto.mediaType.toPhotoBattleKind
        )
    }

    private func editTrip() async throws {
        if let dateError = validateDates() {
            showToast?(dateError)
            return
        }

        if let desc = trip.description, desc.count > 5000 {
            showToast?(PinzBaseStrings.TripInfo.Toast.descriptionTooLong)
            return
        }

        if let uploadTask = tripCoverUploadTask {
            do {
                _ = try await uploadTask.value
            } catch is CancellationError {
                // silent
            } catch let error as MediaUploadError {
                switch error {
                case .limitExceeded(let kind, _, _) where kind == .image:
                    showToast?(MediaUploadPreprocessor.localizedLimitMessage(for: kind))
                default:
                    showToast?(PinzBaseStrings.TripInfo.Toast.coverUploadFailed)
                }
                tripCoverUploadTask = nil
                return
            } catch {
                showToast?(PinzBaseStrings.TripInfo.Toast.coverUploadFailed)
                tripCoverUploadTask = nil
                return
            }
            tripCoverUploadTask = nil
        }

        let nameChanged = editingSnapshot?.name != trip.name
        let descriptionChanged = editingSnapshot?.description != trip.description
        if nameChanged { trip.isNameCensored = false }
        if descriptionChanged { trip.isDescriptionCensored = false }

        do {
            let updatedTrip = try await networkService.updateTrip(
                id: trip.id,
                name: trip.name,
                description: trip.description,
                category: trip.category.apiValue,
                season: trip.season.apiValue,
                privacyLevel: trip.privacyLevel?.lowercased(),
                coverUrl: trip.coverUrl,
                startDateUnix: trip.startDate.flatMap { Int($0.timeIntervalSince1970) },
                endDateUnix: trip.endDate.flatMap { Int($0.timeIntervalSince1970) }
            )
            let oldTripImage = trip.image
            let oldPins = trip.pins
            trip = updatedTrip.toTrip()
            trip.image = oldTripImage
            trip.pins = oldPins
            onTripUpdated?()
            showToast?(PinzBaseStrings.TripInfo.Toast.tripSaved)
            changeState(to: .default)
            editingSnapshot = nil
        } catch {
            showToast?(PinzBaseStrings.TripInfo.Toast.saveFailed)
            throw error
        }
    }

    private func uploadTripCoverTask(with image: UIImage) {
        tripCoverUploadTask?.cancel()

        tripCoverUploadTask = Task { @MainActor [weak self] in
            guard let self else {
                throw CancellationError()
            }
            let response = try await self.uploadTripCoverFlow(image: image)
            self.trip.coverUrl = response.coverUrl
            self.trip.image = nil
            return response
        }
    }

    private func uploadTripCoverFlow(image: UIImage) async throws -> TripDTO {
        let contentType: String

        if let jpegData = image.jpegData(compressionQuality: 0.85) {
            contentType = "image/jpeg"
        } else if let pngData = image.pngData() {
            contentType = "image/png"
        } else {
            throw TripCoverUploadFlowError.missingImageData
        }

        let filename = "cover-\(UUID().uuidString).\(contentType == "image/png" ? "png" : "jpg")"
        let request = try await networkService.requestTripCoverUpload(
            id: trip.id,
            filename: filename,
            contentType: contentType
        )

        guard let uploadUrl = request.uploadUrl,
              let s3Key = request.s3Key,
              !uploadUrl.isEmpty,
              !s3Key.isEmpty else {
            throw TripCoverUploadFlowError.invalidUploadResponse
        }

        let prepared = try await MediaUploadPreprocessor.shared.prepareImage(
            image,
            contentType: contentType,
            uploadURL: uploadUrl,
            context: "trip_cover"
        )
        switch prepared.body {
        case let .data(data):
            try await networkService.uploadToS3(url: uploadUrl, data: data, contentType: prepared.contentType)
        case .file:
            throw TripCoverUploadFlowError.invalidUploadResponse
        }
        return try await networkService.confirmTripCoverUpload(
            id: trip.id,
            s3Key: s3Key
        )
    }

    private func updateNotifications(_ enabled: Bool) async throws {
        do {
            _ = try await networkService.updateTripSettings(id: trip.id, notificationsEnabled: enabled)
            showToast?(PinzBaseStrings.TripInfo.Toast.notificationsUpdated)
        } catch {
            showToast?(PinzBaseStrings.TripInfo.Toast.notificationsFailed)
            throw error
        }
    }

    private func leaveCurrentTrip() async throws {
        do {
            _ = try await networkService.leaveTrip(id: trip.id)
            SelectedTripStorage.shared.clearSelection()
            showToast?(PinzBaseStrings.TripInfo.Toast.tripLeft)
            dispatch(.navigate(.back))
        } catch {
            showToast?(PinzBaseStrings.TripInfo.Toast.leaveFailed)
            throw error
        }
    }

    private func changeState(to state: State) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.state = state
        }
    }

    private func backupTripForEditing() {
        editingSnapshot = trip
    }

    private func restoreTripFromSnapshot() {
        if let snapshot = editingSnapshot {
            trip = snapshot
        }
        editingSnapshot = nil
    }

}
