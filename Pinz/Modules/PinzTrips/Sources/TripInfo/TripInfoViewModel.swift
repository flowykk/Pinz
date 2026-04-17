import SwiftUI
import Foundation
import PinzDomain
import PinzBase
import PinzNetworking

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

    func deleteTrip() async throws {
        try await networkService.deleteTrip(id: trip.id)
        SelectedTripStorage.shared.clearSelection()
    }

    private func editTrip() async throws {
        if let uploadTask = tripCoverUploadTask {
            do {
                _ = try await uploadTask.value
            } catch is CancellationError {
                print("[TripInfo] Trip cover upload canceled")
            } catch {
                print("[TripInfo] Failed to upload trip cover before save: \(error)")
            }
            tripCoverUploadTask = nil
        }

        let updatedTrip = try await networkService.updateTrip(
            id: trip.id,
            name: trip.name,
            description: trip.description,
            category: mapCategory(trip.category),
            season: mapSeason(trip.season),
            privacyLevel: trip.privacyLevel,
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
        changeState(to: .default)
        editingSnapshot = nil
    }

    private func uploadTripCoverTask(with image: UIImage) {
        tripCoverUploadTask?.cancel()

        tripCoverUploadTask = Task { @MainActor [weak self] in
            guard let self else {
                throw CancellationError()
            }
            defer {
                self.tripCoverUploadTask = nil
            }

            let response = try await self.uploadTripCoverFlow(image: image)
            self.trip.coverUrl = response.coverUrl
            self.trip.image = nil
            return response
        }
    }

    private func uploadTripCoverFlow(image: UIImage) async throws -> TripDTO {
        let contentType: String
        let data: Data

        if let jpegData = image.jpegData(compressionQuality: 0.85) {
            contentType = "image/jpeg"
            data = jpegData
        } else if let pngData = image.pngData() {
            contentType = "image/png"
            data = pngData
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

        try await networkService.uploadToS3(url: uploadUrl, data: data, contentType: contentType)
        return try await networkService.confirmTripCoverUpload(
            id: trip.id,
            s3Key: s3Key
        )
    }

    private func updateNotifications(_ enabled: Bool) async throws {
        _ = try await networkService.updateTripSettings(
            id: trip.id,
            notificationsEnabled: enabled
        )
    }

    private func leaveCurrentTrip() async throws {
        _ = try await networkService.leaveTrip(id: trip.id)
        SelectedTripStorage.shared.clearSelection()
        dispatch(.navigate(.back))
    }

    private func mapCategory(_ category: TripCategory) -> String? {
        switch category {
        case .none:
            nil
        case let .custom(value):
            value
        case .vacation:
            "vacation"
        case .holidays:
            "holidays"
        case .business:
            "business"
        case .education:
            "education"
        case .active:
            "active"
        }
    }

    private func mapSeason(_ season: TripSeason) -> String? {
        switch season {
        case .none:
            nil
        case .summer:
            "summer"
        case .autumn:
            "autumn"
        case .winter:
            "winter"
        case .spring:
            "spring"
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
