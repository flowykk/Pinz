import SwiftUI
import MapKit
import PinzDomain
import PinzBase
import PinzNetworking
import PinzPins

@MainActor @Observable
final class TripViewModel {

    public enum Route {
        case tripInfo
        case tripCreation
        case profile(User)
        case feed
        case members
        case pinInfo(Pin)
        case pinCreation
    }

    enum State {
        case `default`
        case route

        mutating func toggle() {
            switch self {
            case .default: self = .route
            case .route: self = .default
            }
        }
    }

    enum Intent {
        case navigate(Route)
        case selectPin(pin: Pin?)
        case unselectPin
        case selectTrip(Trip)
        case checkAndUpdateTrip([Trip])
        case clearSelection
        case forceReloadSavedTrip

        case toggleRouteState
        case nextPin
        case previousPin
    }

    enum AsyncIntent {
        case loadSavedTrip
        case loadCurrentProfile
        case addMedia
        case addPin
    }

    var state: State = .default
    var routePinIndex: Int = 0
    var isLoading: Bool = false
    var isProfileLoading: Bool = false
    var currentUser: User?
    var currentUserAvatarImage: UIImage?
    private var hasLoaded: Bool = false
    private var lastFetchedTripId: String? = nil
    private var shouldReloadSavedTrip: Bool = false
    private var hasLoadedProfile: Bool = false

    var trip: Trip?
    var _position: MapCameraPosition?
    var selectedPin: Pin?
    var hasActivePinUploadSession: Bool = false
    private var participants: [TripParticipantDTO] = []
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?
    private let networkService = NetworkService.shared

    var sortedPins: [Pin] {
        (trip?.pins ?? []).sorted { ($0.startDate ?? .distantPast) < ($1.startDate ?? .distantPast) }
    }

    var position: MapCameraPosition {
        get {
            _position ?? .camera(MapCamera(
                centerCoordinate: CLLocationCoordinate2D(
                    latitude: 55.7558,
                    longitude: 37.6173
                ),
                distance: 50000
            ))
        }
        set { _position = newValue }
    }

    public init(trip: Trip?) {
        self.trip = trip
        if let trip {
            self._position = trip.pins.calculateInitialMapPosition()
        }
    }
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .tripInfo:
                if let trip {
                    router?.navigateToTripInfo(trip: trip) { [weak self] in
                        guard let self else {
                            return
                        }
                        self.dispatch(.forceReloadSavedTrip)
                        Task {
                            try? await self.asyncDispatch(.loadSavedTrip)
                        }
                    }
                }
            case .tripCreation:
                router?.navigateToTripCreationInitial()
            case .profile(let user):
                router?.navigateToProfile(user: user)
            case .feed:
                router?.navigateToFeed()
            case .pinInfo(let pin):
                print("[TripViewModel] navigating to PinInfo pin=\(pin.serverId ?? "nil") isPrivate=\(pin.isPrivate)")
                for media in pin.medias {
                    print("[TripViewModel]   media \(media.mediaId ?? "nil") isPrivate=\(media.isPrivate)")
                }
                router?.navigateToPinInfo(
                    pin: pin,
                    updateAction: PinUpdateAction { [weak self] updatedPin in
                        guard let self, let idx = trip?.pins.firstIndex(where: { $0.serverId == updatedPin.serverId }) else { return }
                        trip?.pins[idx] = updatedPin
                    },
                    deleteAction: PinDeleteAction { [weak self] deletedPin in
                        guard let self else { return }
                        trip?.pins.removeAll { $0.serverId == deletedPin.serverId }
                    }
                )
            case .pinCreation:
                router?.navigateToPinCreation()
            case .members:
                guard let trip else { return }
                router?.navigateToTripMembers(
                    tripId: trip.id,
                    participants: participants,
                    currentUserId: currentUser?.profileId
                )
            }
        case let .selectPin(pin):
            selectedPin = pin
        case .unselectPin:
            if let selectedPin {
                dispatch(.navigate(.pinInfo(selectedPin)))
            }
            selectedPin = nil
        case let .selectTrip(trip):
            self.trip = trip
            position = trip.pins.calculateInitialMapPosition()
            selectedPin = nil
            state = .default
            routePinIndex = 0
            SelectedTripStorage.shared.selectTrip(id: trip.id)
        case let .checkAndUpdateTrip(trips):
            guard let selectedTripID = SelectedTripStorage.shared.selectedTripID,
                  selectedTripID != trip?.id,
                  let newTrip = trips.first(where: { $0.id == selectedTripID }) else {
                return
            }
            dispatch(.selectTrip(newTrip))
        case .clearSelection:
            trip = nil
            lastFetchedTripId = nil
            hasLoaded = false
            isLoading = false
            routePinIndex = 0
            state = .default
            selectedPin = nil
            _position = nil
            shouldReloadSavedTrip = false
            SelectedTripStorage.shared.clearSelection()

        case .forceReloadSavedTrip:
            shouldReloadSavedTrip = true
        case .toggleRouteState:
            withAnimation(.easeInOut(duration: 0.3)) {
                state.toggle()
            }
            routePinIndex = 0
            if state == .route {
                navigateToRoutePin(at: 0)
            }
        case .nextPin:
            guard routePinIndex < sortedPins.count - 1 else { return }
            routePinIndex += 1
            navigateToRoutePin(at: routePinIndex)
        case .previousPin:
            guard routePinIndex > 0 else { return }
            routePinIndex -= 1
            navigateToRoutePin(at: routePinIndex)
        }
    }
    
    public func setRouter(_ router: AppRouting?) {
        self.router = router
        router?.clearCurrentProfileUpdates()
        router?.subscribeToCurrentProfileUpdates { [weak self] updatedUser in
            Task { @MainActor in
                await self?.applyProfileUpdateFromProfileScreen(updatedUser)
            }
        }
        refreshActivePinUploadSessionFlag()
    }

    public func setShowToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    public func refreshActivePinUploadSessionFlag() {
        guard let tripId = trip?.id else {
            hasActivePinUploadSession = false
            return
        }
        hasActivePinUploadSession = PinUploadSessionStorage.shared.sessionId(forTripId: tripId) != nil
    }

    private func navigateToRoutePin(at index: Int) {
        let pins = sortedPins
        guard !pins.isEmpty, index >= 0, index < pins.count else { return }

        let direction: Int
        if index > routePinIndex {
            direction = 1
        } else if index < routePinIndex {
            direction = -1
        } else {
            direction = 0
        }

        func setCamera(at index: Int) {
            guard let coordinate = pins[index].coordinates else {
                return
            }
            routePinIndex = index
            withAnimation(.easeInOut(duration: 1)) {
                position = .camera(
                    MapCamera(
                        centerCoordinate: coordinate,
                        distance: 5000
                    )
                )
            }
        }

        var targetIndex = index
        if pins[targetIndex].coordinates != nil {
            setCamera(at: targetIndex)
            return
        }

        if direction == 0 {
            var offset = 1
            while offset < pins.count {
                let nextIndex = targetIndex + offset
                if nextIndex < pins.count, pins[nextIndex].coordinates != nil {
                    setCamera(at: nextIndex)
                    return
                }

                let previousIndex = targetIndex - offset
                if previousIndex >= 0, pins[previousIndex].coordinates != nil {
                    setCamera(at: previousIndex)
                    return
                }

                offset += 1
            }

            return
        }

        while true {
            let nextIndex = targetIndex + direction
            guard nextIndex >= 0, nextIndex < pins.count else {
                return
            }
            targetIndex = nextIndex
            if pins[targetIndex].coordinates != nil {
                setCamera(at: targetIndex)
                return
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .loadSavedTrip:
            guard let tripId = SelectedTripStorage.shared.selectedTripID else {
                return
            }
            let shouldReload = shouldReloadSavedTrip
            guard shouldReload || lastFetchedTripId != tripId else {
                return
            }
            if shouldReload || !hasLoaded {
                withAnimation { isLoading = true }
            }
            defer {
                if isLoading {
                    withAnimation { isLoading = false }
                }
            }
            do {
                let response = try await networkService.getTrip(id: tripId)
                var trip = response.trip.toTrip()
                if let coverUrl = trip.coverUrl {
                    trip.image = await ImageProvider.loadOrGetImage(
                        for: coverUrl,
                        .group,
                        cacheVariant: .thumbnail,
                        targetPixel: 560
                    )
                }
                trip.pins = response.pins.enumerated().map { index, dto in
                    dto.toPin(
                        index: index,
                        tripId: trip.id,
                        nameIfMissing: PinzBaseStrings.Common.Label.pinNumber(index + 1)
                    )
                }
                participants = response.participants
                lastFetchedTripId = tripId
                dispatch(.selectTrip(trip))
                shouldReloadSavedTrip = false
                hasLoaded = true
            } catch {
                throw error
            }
        case .addMedia:
            guard let tripId = trip?.id else { return }
            let response = try await networkService.getTrip(id: tripId)
            let sessionId = response.activeAddMediaSession?.sessionId
            switch response.trip.status ?? "" {
            case "READY":
                router?.navigateToAddMediaStart(tripId: tripId)
            case "ADD_MEDIA_UPLOADING":
                if let sessionId { router?.navigateToAddMediaUploading(tripId: tripId, sessionId: sessionId) }
            case "ADD_MEDIA_GROUPING_REVIEW":
                if let sessionId { router?.navigateToAddMediaGrouping(tripId: tripId, sessionId: sessionId) }
            case "ADD_MEDIA_PROCESSING":
                if let sessionId { router?.navigateToAddMediaProcessing(tripId: tripId, sessionId: sessionId) }
            case "ADD_MEDIA_DRAFT_FINAL_REVIEW":
                if let sessionId { router?.navigateToAddMediaReview(tripId: tripId, sessionId: sessionId) }
            default:
                break
            }

        case .addPin:
            guard let tripId = trip?.id else { return }
            await PinUploadEntryResolver.resume(
                tripId: tripId,
                networkService: networkService,
                router: router,
                showToast: showToast
            )
            refreshActivePinUploadSessionFlag()

        case .loadCurrentProfile:
            guard !hasLoadedProfile else {
                return
            }

            isProfileLoading = true
            defer {
                isProfileLoading = false
            }

            do {
                let response = try await networkService.getProfile()
                let loadedUser = response.toUser()
                currentUser = loadedUser
                await loadCurrentUserAvatar(for: loadedUser)
                hasLoadedProfile = true
            } catch {
                print("[TripView] Failed to load profile: \(error)")
            }
        }
    }

    private func applyProfileUpdateFromProfileScreen(_ updatedUser: User) async {
        currentUser = updatedUser
        await refreshCurrentUserAvatar(for: updatedUser.avatarUrl)
    }

    private func refreshCurrentUserAvatar(for avatarUrl: String?) async {
        if let avatarUrl, !avatarUrl.isEmpty {
            FileManagerImageStorage.shared.deleteImage(url: avatarUrl)
        }

        currentUserAvatarImage = await ImageProvider.loadOrGetImage(
            for: avatarUrl,
            .user,
            cacheVariant: .thumbnail,
            targetPixel: 120
        )
    }

    private func loadCurrentUserAvatar(for user: User) async {
        await refreshCurrentUserAvatar(for: user.avatarUrl)
    }
}
