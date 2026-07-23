// swiftlint:disable file_length
import Foundation
import PinzNetworking
import PinzDomain

// swiftlint:disable:next type_body_length
final class MockNetworkService: NetworkServiceProtocol {

    // MARK: - Auth

    var devLoginResult: Result<UserTokensDTO, Error> = .success(UserTokensDTO(accessToken: "access", refreshToken: "refresh"))
    var submitEmailResult: Result<SubmitEmailDTO, Error> = .success(SubmitEmailDTO(isRegistered: false, registrationId: "reg-123"))
    var verifyEmailResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var passkeyLoginBeginResult: Result<PasskeyOptionsDTO, Error> = .success(PasskeyOptionsDTO(optionsJson: ""))
    var passkeyLoginFinishResult: Result<UserTokensDTO, Error> = .success(UserTokensDTO(accessToken: "access", refreshToken: "refresh"))
    var passkeyRegisterBeginResult: Result<PasskeyOptionsDTO, Error> = .success(PasskeyOptionsDTO(optionsJson: ""))
    var passkeyRegisterFinishResult: Result<UserTokensDTO, Error> = .success(UserTokensDTO(accessToken: "access", refreshToken: "refresh"))
    var refreshTokenResult: Result<RefreshTokenDTO, Error> = .success(RefreshTokenDTO(accessToken: "access"))
    var logoutResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))

    // MARK: - Feed / Trips

    var getFeedResult: Result<[FeedItemDTO], Error> = .success(MockNetworkService.stubFeed)
    var getRecommendationsResult: Result<GetRecommendationsResponseDTO, Error> = .success(MockNetworkService.stubGetRecommendationsResponse)
    var saveRecommendationResult: Result<SaveRecommendationResponseDTO, Error> = .success(MockNetworkService.stubSaveRecommendationResponse)
    var getRecommendationsCall: (city: String?, country: String?, category: String?, season: String?)?
    var saveRecommendationCall: (
        snapshotToken: String,
        pinIds: [String],
        city: String?,
        country: String?,
        category: String?,
        season: String?
    )?
    var getTripsResult: Result<[TripDTO], Error> = .success([])
    var getTripResult: Result<GetTripResponseDTO, Error> = .success(MockNetworkService.stubTripResponse)
    var requestTripCoverUploadResult: Result<TripCoverUploadResponseDTO, Error> = .success(
        TripCoverUploadResponseDTO(
            uploadUrl: "https://example.com/upload",
            s3Key: "mock-trip-s3-key"
        )
    )
    var confirmTripCoverUploadResult: Result<TripDTO, Error> = .success(MockNetworkService.stubTrip)
    var updateTripResult: Result<TripDTO, Error> = .success(MockNetworkService.stubTrip)
    var updateTripCall: UpdateTripCall?
    var deleteTripError: Error?
    var requestTripCoverUploadCall: (id: String, filename: String, contentType: String)?
    var confirmTripCoverUploadCall: (id: String, s3Key: String)?

    var joinTripByTokenResult: Result<JoinTripByTokenDTO, Error> = .success(JoinTripByTokenDTO(tripId: "trip-001", alreadyJoined: false))
    var generateInviteLinkResult: Result<GenerateInviteLinkDTO, Error> = .success(
        GenerateInviteLinkDTO(inviteLinkId: "link-001", inviteUrl: "https://pinz.website/join/token", token: "token", expiresAtUnix: nil)
    )
    var generateInviteLinkCallCount = 0
    var lastGenerateInviteLinkTripId: String?
    var lastGenerateInviteExpires: Int?
    var leaveTripResult: Result<LeaveTripDTO, Error> = .success(LeaveTripDTO(success: true, tripDeleted: false))
    var leaveTripCall: String?
    var removeParticipantError: Error?
    var publishTripResult: Result<TripDTO, Error> = .success(MockNetworkService.stubTrip)
    var updateTripSettingsResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var updateTripSettingsCall: (id: String, notificationsEnabled: Bool)?
    var likeTripResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var dislikeTripResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var addTripToFavouritesResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var likeTripCall: String?
    var dislikeTripCall: String?
    var addTripToFavouritesCall: String?
    var removeTripFromFavouritesCall: String?
    var getFavouriteTripsResult: Result<[TripDTO], Error> = .success([])
    var removeTripFromFavouritesError: Error?

    // MARK: - Profile

    var getProfileResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var updateProfileResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var deleteAvatarResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var deleteAccountResult: Result<DeleteAccountResponseDTO, Error> = .success(DeleteAccountResponseDTO(success: true))
    var getVisitedLocationsResult: Result<VisitedLocationsResponseDTO, Error> = .success(VisitedLocationsResponseDTO())
    var getVisitedLocationsCountryResult: Result<VisitedLocationsResponseDTO, Error>?
    var getVisitedLocationsCityResult: Result<VisitedLocationsResponseDTO, Error>?
    var getVisitedLocationsCallTypes: [String] = []
    var getProfileStatsResult: Result<UserStatsResponseDTO, Error> = .success(
        UserStatsResponseDTO(totalTrips: 0, totalPins: 0, totalMedia: 0, totalLikes: 0, totalDislikes: 0, battlesFinished: 0)
    )
    var getProfileStatsCallCount = 0
    var requestAvatarUploadResult: Result<AvatarUploadResponseDTO, Error> = .success(
        AvatarUploadResponseDTO(uploadUrl: "https://example.com/upload", s3Key: "mock-s3-key")
    )
    var confirmAvatarUploadResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var changeEmailResult: Result<ChangeEmailResponseDTO, Error> = .success(ChangeEmailResponseDTO(success: true))
    var changeEmailCall: (userId: String?, newEmail: String)?
    var confirmEmailChangeResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var confirmEmailChangeCall: String?
    var registerDeviceTokenResult: Result<DeviceTokenRegisterResponseDTO, Error> = .success(
        DeviceTokenRegisterResponseDTO(tokenId: "550e8400-e29b-41d4-a716-446655440000")
    )
    var unregisterDeviceTokenResult: Result<DeviceTokenUnregisterResponseDTO, Error> = .success(
        DeviceTokenUnregisterResponseDTO(success: true)
    )
    var registerDeviceTokenCall: String?
    var unregisterDeviceTokenCall: String?
    var requestAvatarUploadCall: (filename: String, contentType: String)?
    var confirmAvatarUploadCall: String?
    var uploadToS3Call: (url: String, dataBytes: Int, contentType: String)?
    var uploadToS3FileURLCall: (url: String, fileURL: String, contentType: String)?

    // MARK: - Trip creation / add-media

    var createTripResult: Result<CreateTripDTO, Error> = .success(CreateTripDTO(tripId: "trip-001", status: "created", uploadUrls: []))
    var processMediaGroupingResult: Result<ProcessMediaGroupingDTO, Error> = .success(
        ProcessMediaGroupingDTO(tripId: "trip-001", status: "processed", draftPins: [])
    )
    var applyGroupsAndProcessResult: Result<ApplyGroupsAndProcessDTO, Error> = .success(
        ApplyGroupsAndProcessDTO(message: "processing", status: "ok")
    )
    var waitForTripProcessingCompletedResult: Result<Void, Error> = .success(())
    var getTripReviewResult: Result<GetTripReviewDTO, Error> = .success(
        GetTripReviewDTO(tripId: "trip-001", status: "ready", pins: [], similar: [])
    )
    var finalizeTripResult: Result<FinalizeTripDTO, Error> = .success(
        FinalizeTripDTO(tripId: "trip-001", status: "finalized", message: "done")
    )
    var startBattleResult: Result<StartBattleResponseDTO, Error> = .success(
        StartBattleResponseDTO(
            battleId: "battle-001",
            media: MockNetworkService.stubBattleMedia()
        )
    )
    var submitBattleResultResult: Result<SubmitBattleResultResponseDTO, Error> = .success(
        SubmitBattleResultResponseDTO(success: true, newBattleRating: 1)
    )
    var startBattleCall: String?
    var submitBattleResultCall: (tripId: String, battleId: String, winnerMediaId: String)?
    var uploadToS3Error: Error?

    // MARK: - Auth implementations

    func devLogin(email: String) async throws -> UserTokensDTO { try devLoginResult.get() }
    func submitEmail(email: String) async throws -> SubmitEmailDTO { try submitEmailResult.get() }
    func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessDTO { try verifyEmailResult.get() }
    func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsDTO { try passkeyLoginBeginResult.get() }
    func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensDTO { try passkeyLoginFinishResult.get() }
    func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsDTO { try passkeyRegisterBeginResult.get() }
    func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensDTO { try passkeyRegisterFinishResult.get() }
    func refreshToken(refreshToken: String) async throws -> RefreshTokenDTO { try refreshTokenResult.get() }
    func logout(refreshToken: String) async throws -> SuccessDTO { try logoutResult.get() }

    // MARK: - Feed

    func getFeed(limit: Int?, offset: Int?, category: String?, season: String?, city: String?, country: String?, sortBy: String?) async throws -> [FeedItemDTO] {
        try getFeedResult.get()
    }

    func getRecommendations(
        city: String?,
        country: String?,
        category: String?,
        season: String?
    ) async throws -> GetRecommendationsResponseDTO {
        getRecommendationsCall = (city, country, category, season)
        return try getRecommendationsResult.get()
    }

    func saveRecommendation(
        snapshotToken: String,
        pinIds: [String],
        city: String?,
        country: String?,
        category: String?,
        season: String?
    ) async throws -> SaveRecommendationResponseDTO {
        saveRecommendationCall = (snapshotToken, pinIds, city, country, category, season)
        return try saveRecommendationResult.get()
    }

    // MARK: - Trips CRUD

    func getTrips() async throws -> [TripDTO] { try getTripsResult.get() }
    func getTrip(id: String) async throws -> GetTripResponseDTO { try getTripResult.get() }
    func requestTripCoverUpload(id: String, filename: String, contentType: String) async throws -> TripCoverUploadResponseDTO {
        requestTripCoverUploadCall = (id, filename, contentType)
        return try requestTripCoverUploadResult.get()
    }
    func confirmTripCoverUpload(id: String, s3Key: String) async throws -> TripDTO {
        confirmTripCoverUploadCall = (id, s3Key)
        return try confirmTripCoverUploadResult.get()
    }
    func updateTrip(id: String, name: String?, description: String?, category: String?, season: String?, privacyLevel: String?, coverUrl: String?, startDateUnix: Int?, endDateUnix: Int?) async throws -> TripDTO {
        updateTripCall = UpdateTripCall(
            id: id,
            name: name,
            description: description,
            category: category,
            season: season,
            privacyLevel: privacyLevel,
            coverUrl: coverUrl,
            startDateUnix: startDateUnix,
            endDateUnix: endDateUnix
        )
        return try updateTripResult.get()
    }
    func deleteTrip(id: String) async throws {
        if let error = deleteTripError { throw error }
    }

    // MARK: - Trip actions

    func joinTripByToken(token: String) async throws -> JoinTripByTokenDTO { try joinTripByTokenResult.get() }
    func generateInviteLink(tripId: String, expiresInSeconds: Int?) async throws -> GenerateInviteLinkDTO {
        generateInviteLinkCallCount += 1
        lastGenerateInviteLinkTripId = tripId
        lastGenerateInviteExpires = expiresInSeconds
        return try generateInviteLinkResult.get()
    }
    func leaveTrip(id: String) async throws -> LeaveTripDTO {
        leaveTripCall = id
        return try leaveTripResult.get()
    }
    func removeParticipant(tripId: String, userId: String) async throws {
        if let error = removeParticipantError { throw error }
    }
    var publishTripCall: (id: String, publishWhole: Bool, pinIds: [String])?
    func publishTrip(id: String, publishWhole: Bool, pinIds: [String]) async throws -> TripDTO {
        publishTripCall = (id, publishWhole, pinIds)
        return try publishTripResult.get()
    }
    func updateTripSettings(id: String, notificationsEnabled: Bool) async throws -> SuccessDTO {
        updateTripSettingsCall = (id: id, notificationsEnabled: notificationsEnabled)
        return try updateTripSettingsResult.get()
    }
    func likeTrip(id: String) async throws -> SuccessDTO {
        likeTripCall = id
        return try likeTripResult.get()
    }
    func dislikeTrip(id: String) async throws -> SuccessDTO {
        dislikeTripCall = id
        return try dislikeTripResult.get()
    }
    func addTripToFavourites(id: String) async throws -> SuccessDTO {
        addTripToFavouritesCall = id
        return try addTripToFavouritesResult.get()
    }
    func removeTripFromFavourites(id: String) async throws {
        removeTripFromFavouritesCall = id
        if let error = removeTripFromFavouritesError { throw error }
    }
    func getProfile() async throws -> ProfileResponseDTO { try getProfileResult.get() }
    func getVisitedLocations(type: String?) async throws -> VisitedLocationsResponseDTO {
        if let type { getVisitedLocationsCallTypes.append(type) }
        if type == "country", let result = getVisitedLocationsCountryResult { return try result.get() }
        if type == "city", let result = getVisitedLocationsCityResult { return try result.get() }
        return try getVisitedLocationsResult.get()
    }
    func getProfileStats() async throws -> UserStatsResponseDTO {
        getProfileStatsCallCount += 1
        return try getProfileStatsResult.get()
    }
    var updateProfileCall: String?
    func updateProfile(username: String) async throws -> ProfileResponseDTO {
        updateProfileCall = username
        return try updateProfileResult.get()
    }
    func deleteAccount() async throws -> DeleteAccountResponseDTO { try deleteAccountResult.get() }
    func deleteAvatar() async throws -> ProfileResponseDTO { try deleteAvatarResult.get() }
    func requestAvatarUpload(filename: String, contentType: String) async throws -> AvatarUploadResponseDTO {
        requestAvatarUploadCall = (filename, contentType)
        return try requestAvatarUploadResult.get()
    }
    func confirmAvatarUpload(s3Key: String) async throws -> ProfileResponseDTO {
        confirmAvatarUploadCall = s3Key
        return try confirmAvatarUploadResult.get()
    }
    func changeEmail(userId: String?, newEmail: String) async throws -> ChangeEmailResponseDTO {
        changeEmailCall = (userId, newEmail)
        return try changeEmailResult.get()
    }
    func confirmEmailChange(verificationCode: String) async throws -> ProfileResponseDTO {
        confirmEmailChangeCall = verificationCode
        return try confirmEmailChangeResult.get()
    }

    func registerDeviceToken(apnsToken: String) async throws -> DeviceTokenRegisterResponseDTO {
        registerDeviceTokenCall = apnsToken
        return try registerDeviceTokenResult.get()
    }

    func unregisterDeviceToken(apnsToken: String) async throws -> DeviceTokenUnregisterResponseDTO {
        unregisterDeviceTokenCall = apnsToken
        return try unregisterDeviceTokenResult.get()
    }
    func getFavouriteTrips(limit: Int?, offset: Int?) async throws -> [TripDTO] {
        try getFavouriteTripsResult.get()
    }

    func uploadToS3(url: String, data: Data, contentType: String) async throws {
        uploadToS3Call = (url, data.count, contentType)
        if let uploadToS3Error { throw uploadToS3Error }
    }

    func uploadToS3(url: String, fileURL: URL, contentType: String) async throws {
        uploadToS3FileURLCall = (url, fileURL.absoluteString, contentType)
        if let uploadToS3Error { throw uploadToS3Error }
    }

    // MARK: - Trip creation flow

    func createTrip(name: String, description: String?, category: String?, season: String?, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO { try createTripResult.get() }
    func processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO { try processMediaGroupingResult.get() }
    func applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO { try applyGroupsAndProcessResult.get() }
    func waitForTripProcessingCompleted(tripId: String, timeout: TimeInterval) async throws {
        _ = try waitForTripProcessingCompletedResult.get()
    }
    func getTripReview(tripId: String) async throws -> GetTripReviewDTO { try getTripReviewResult.get() }
    func finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String]) async throws -> FinalizeTripDTO { try finalizeTripResult.get() }
    func startBattle(tripId: String) async throws -> StartBattleResponseDTO {
        startBattleCall = tripId
        return try startBattleResult.get()
    }
    func submitBattleResult(tripId: String, battleId: String, winnerMediaId: String) async throws -> SubmitBattleResultResponseDTO {
        submitBattleResultCall = (tripId: tripId, battleId: battleId, winnerMediaId: winnerMediaId)
        return try submitBattleResultResult.get()
    }

    // MARK: - Public profile

    var getPublicUserProfileResult: Result<PublicUserProfileResponseDTO, Error> = .success(
        PublicUserProfileResponseDTO(id: "user-public", username: "publicUser", avatarUrl: nil, createdAt: 0, desiredPlaces: [])
    )
    func getPublicUserProfile(id: String) async throws -> PublicUserProfileResponseDTO { try getPublicUserProfileResult.get() }

    // MARK: - Privacy

    var setTripPrivacyResult: Result<PrivacyResponseDTO, Error> = .success(PrivacyResponseDTO(privacyLevel: "public"))
    var setPinPrivacyResult: Result<PrivacyResponseDTO, Error> = .success(PrivacyResponseDTO(privacyLevel: "public"))
    var setMediaPrivacyResult: Result<PrivacyResponseDTO, Error> = .success(PrivacyResponseDTO(privacyLevel: "public"))
    func setTripPrivacy(tripId: String, privacyLevel: String) async throws -> PrivacyResponseDTO { try setTripPrivacyResult.get() }
    func setPinPrivacy(tripId: String, pinId: String, privacyLevel: String) async throws -> PrivacyResponseDTO { try setPinPrivacyResult.get() }
    func setMediaPrivacy(tripId: String, mediaId: String, privacyLevel: String) async throws -> PrivacyResponseDTO { try setMediaPrivacyResult.get() }

    // MARK: - Desired Places

    var getDesiredPlacesResult: Result<[DesiredPlaceDTO], Error> = .success([])
    var createDesiredPlaceResult: Result<DesiredPlaceDTO, Error> = .success(MockNetworkService.stubDesiredPlace)
    var requestDesiredPlaceImageUploadResult: Result<DesiredPlaceImageUploadResponseDTO, Error> = .success(
        DesiredPlaceImageUploadResponseDTO(uploadUrl: "https://s3.example.com/upload", s3Key: "places/mock.jpg")
    )
    var updateDesiredPlaceResult: Result<DesiredPlaceDTO, Error> = .success(MockNetworkService.stubDesiredPlace)
    var deleteDesiredPlaceError: Error?
    var deleteDesiredPlaceImageResult: Result<DesiredPlaceDTO, Error> = .success(MockNetworkService.stubDesiredPlace)

    var createDesiredPlaceCall: (name: String, description: String, s3Key: String?)?
    var updateDesiredPlaceCall: (placeId: String, name: String, description: String, imageS3Key: String?)?
    var deleteDesiredPlaceCall: String?
    var requestDesiredPlaceImageUploadCall: (filename: String, contentType: String)?

    func getDesiredPlaces() async throws -> [DesiredPlaceDTO] { try getDesiredPlacesResult.get() }
    func createDesiredPlace(name: String, description: String, s3Key: String?) async throws -> DesiredPlaceDTO {
        createDesiredPlaceCall = (name, description, s3Key)
        return try createDesiredPlaceResult.get()
    }
    func requestDesiredPlaceImageUpload(filename: String, contentType: String) async throws -> DesiredPlaceImageUploadResponseDTO {
        requestDesiredPlaceImageUploadCall = (filename, contentType)
        return try requestDesiredPlaceImageUploadResult.get()
    }
    func updateDesiredPlace(placeId: String, name: String, description: String, imageS3Key: String?) async throws -> DesiredPlaceDTO {
        updateDesiredPlaceCall = (placeId, name, description, imageS3Key)
        return try updateDesiredPlaceResult.get()
    }
    func deleteDesiredPlace(placeId: String) async throws -> SuccessDTO {
        deleteDesiredPlaceCall = placeId
        if let deleteDesiredPlaceError { throw deleteDesiredPlaceError }
        return SuccessDTO(success: true)
    }
    func deleteDesiredPlaceImage(placeId: String) async throws -> DesiredPlaceDTO { try deleteDesiredPlaceImageResult.get() }

    // MARK: - Pins

    var getPinResult: Result<PinResponseDTO, Error> = .success(
        PinResponseDTO(pin: MockNetworkService.stubTripPins[0])
    )
    var updatePinResult: Result<PinResponseDTO, Error> = .success(
        PinResponseDTO(pin: MockNetworkService.stubTripPins[0])
    )
    var deletePinResult: Result<DeletePinResponseDTO, Error> = .success(
        DeletePinResponseDTO(deletionMode: "full")
    )
    var searchPinsResult: Result<[TripPinDTO], Error> = .success([])

    // MARK: - Pin upload

    var pinUploadStartResult: Result<PinUploadStartResponseDTO, Error> = .success(
        PinUploadStartResponseDTO(sessionId: "mock-pin-upload-session", uploadUrls: [])
    )
    var pinUploadRequestUploadUrlsResult: Result<[UploadURLDTO], Error> = .success([])
    var pinUploadCommitUploadResult: Result<PinUploadCommitResponseDTO, Error> = .success(
        PinUploadCommitResponseDTO(mediaId: "mock-media", mediaCountInSession: 1)
    )
    var pinUploadProcessResult: Result<PinUploadProcessResponseDTO, Error> = .success(
        PinUploadProcessResponseDTO(sessionId: "mock-pin-upload-session", processingStatus: "PROCESSING")
    )
    var pinUploadGetReviewResult: Result<PinUploadReviewResponseDTO, Error> = .success(
        PinUploadReviewResponseDTO(
            sessionId: "mock-pin-upload-session",
            processingStatus: "READY_FOR_REVIEW",
            draft: nil,
            similar: nil
        )
    )
    var pinUploadFinalizeResult: Result<PinResponseDTO, Error> = .success(
        PinResponseDTO(pin: MockNetworkService.stubTripPins[0])
    )
    var pinUploadCancelError: Error?

    var addMediaStartResult: Result<AddMediaStartDTO, Error> = .success(
        AddMediaStartDTO(sessionId: "mock-add-media-session", status: "ADD_MEDIA_UPLOADING", joined: false, uploadUrls: [])
    )
    var addMediaRequestUploadUrlsResult: Result<[UploadURLDTO], Error> = .success([])
    var addMediaCommitUploadResult: Result<AddMediaCommitUploadDTO, Error> = .success(
        AddMediaCommitUploadDTO(mediaId: "mock-add-media", mediaCountInSession: 1, remainingSlots: 99)
    )
    var addMediaGetSessionMediaResult: Result<AddMediaSessionMediaDTO, Error> = .success(
        AddMediaSessionMediaDTO(sessionId: "mock-add-media-session", media: [], mediaCountInSession: 0)
    )
    var addMediaGroupingResult: Result<AddMediaGroupingDTO, Error> = .success(
        AddMediaGroupingDTO(
            tripId: "trip-001",
            sessionId: "mock-add-media-session",
            status: "ADD_MEDIA_GROUPING_REVIEW",
            draftPins: []
        )
    )
    var addMediaGetReviewResult: Result<AddMediaReviewDTO, Error> = .success(
        AddMediaReviewDTO(
            tripId: "trip-001",
            sessionId: "mock-add-media-session",
            pins: MockNetworkService.stubTripPins,
            newPinIds: [],
            protectedMediaIds: [],
            canEdit: true
        )
    )
    var addMediaConfirmResult: Result<AddMediaConfirmDTO, Error> = .success(
        AddMediaConfirmDTO(status: "READY", alreadyConfirmed: false)
    )
    var addMediaCancelError: Error?
    var addMediaTakeoverResult: Result<AddMediaTakeoverDTO, Error> = .success(
        AddMediaTakeoverDTO(isInitiator: true)
    )

    var pinUploadGetReviewCall: (tripId: String, sessionId: String)?

    var getPinCall: (tripId: String, pinId: String)?
    var updatePinCall: (
        tripId: String,
        pinId: String,
        name: String?,
        description: String?,
        category: String?,
        latitude: Double?,
        longitude: Double?,
        startTimeUnix: Int?,
        endTimeUnix: Int?,
        tags: [String]?,
        tagsSet: Bool?
    )?
    var deletePinCall: (tripId: String, pinId: String)?
    var deletePinMediaCall: (tripId: String, pinId: String, mediaId: String)?
    var deletePinMediaResult: Result<PinResponseDTO, Error> = .success(
        PinResponseDTO(pin: MockNetworkService.stubTripPins[0])
    )

    func getPin(tripId: String, pinId: String) async throws -> PinResponseDTO {
        getPinCall = (tripId, pinId)
        return try getPinResult.get()
    }

    func updatePin(
        tripId: String,
        pinId: String,
        name: String?,
        description: String?,
        category: String?,
        latitude: Double?,
        longitude: Double?,
        startTimeUnix: Int?,
        endTimeUnix: Int?,
        tags: [String]?,
        tagsSet: Bool?
    ) async throws -> PinResponseDTO {
        updatePinCall = (tripId, pinId, name, description, category, latitude, longitude, startTimeUnix, endTimeUnix, tags, tagsSet)
        return try updatePinResult.get()
    }

    func deletePin(tripId: String, pinId: String) async throws -> DeletePinResponseDTO {
        deletePinCall = (tripId, pinId)
        return try deletePinResult.get()
    }

    func deletePinMedia(tripId: String, pinId: String, mediaId: String) async throws -> PinResponseDTO {
        deletePinMediaCall = (tripId, pinId, mediaId)
        return try deletePinMediaResult.get()
    }

    func searchPins(q: String, limit: Int?, offset: Int?) async throws -> [TripPinDTO] {
        try searchPinsResult.get()
    }

    func pinUploadStart(
        tripId: String,
        targetPinId: String?,
        filesToUpload: [FileToUploadDTO]
    ) async throws -> PinUploadStartResponseDTO {
        try pinUploadStartResult.get()
    }

    func pinUploadRequestUploadUrls(
        tripId: String,
        sessionId: String,
        filesToUpload: [FileToUploadDTO]
    ) async throws -> [UploadURLDTO] {
        try pinUploadRequestUploadUrlsResult.get()
    }

    func pinUploadCommitUpload(
        tripId: String,
        sessionId: String,
        s3Key: String,
        mediaType: String,
        capturedAtUnix: Int?,
        latitude: Double?,
        longitude: Double?
    ) async throws -> PinUploadCommitResponseDTO {
        try pinUploadCommitUploadResult.get()
    }

    func pinUploadProcess(tripId: String, sessionId: String) async throws -> PinUploadProcessResponseDTO {
        try pinUploadProcessResult.get()
    }

    func pinUploadGetReview(tripId: String, sessionId: String) async throws -> PinUploadReviewResponseDTO {
        pinUploadGetReviewCall = (tripId, sessionId)
        return try pinUploadGetReviewResult.get()
    }

    func pinUploadFinalize(
        tripId: String,
        sessionId: String,
        input: PinUploadFinalizeInputDTO
    ) async throws -> PinResponseDTO {
        try pinUploadFinalizeResult.get()
    }

    func pinUploadCancel(tripId: String, sessionId: String) async throws {
        if let pinUploadCancelError { throw pinUploadCancelError }
    }

    func addMediaStart(tripId: String, filesToUpload: [FileToUploadDTO]) async throws -> AddMediaStartDTO {
        try addMediaStartResult.get()
    }

    func addMediaRequestUploadUrls(
        tripId: String,
        sessionId: String,
        filesToUpload: [FileToUploadDTO]
    ) async throws -> [UploadURLDTO] {
        try addMediaRequestUploadUrlsResult.get()
    }

    func addMediaCommitUpload(
        tripId: String,
        sessionId: String,
        s3Key: String,
        mediaType: String,
        capturedAt: String?,
        latitude: Double?,
        longitude: Double?
    ) async throws -> AddMediaCommitUploadDTO {
        try addMediaCommitUploadResult.get()
    }

    func addMediaGetSessionMedia(tripId: String, sessionId: String) async throws -> AddMediaSessionMediaDTO {
        try addMediaGetSessionMediaResult.get()
    }

    func addMediaProcessGrouping(tripId: String, sessionId: String, addMore: Bool) async throws -> AddMediaGroupingDTO {
        try addMediaGroupingResult.get()
    }

    func addMediaGetGrouping(tripId: String, sessionId: String) async throws -> AddMediaGroupingDTO {
        try addMediaGroupingResult.get()
    }

    func addMediaApplyGroupsAndProcess(
        tripId: String,
        sessionId: String,
        draftPins: [DraftPinInputDTO],
        deletedMediaIds: [String]
    ) async throws -> ApplyGroupsAndProcessDTO {
        ApplyGroupsAndProcessDTO(message: "stub", status: "ADD_MEDIA_DRAFT_FINAL_REVIEW")
    }

    func addMediaGetReview(tripId: String, sessionId: String) async throws -> AddMediaReviewDTO {
        try addMediaGetReviewResult.get()
    }

    func addMediaConfirm(
        tripId: String,
        sessionId: String,
        pinUpdates: [PinUpdateInputDTO],
        mediaToDelete: [String]
    ) async throws -> AddMediaConfirmDTO {
        try addMediaConfirmResult.get()
    }

    func addMediaCancel(tripId: String, sessionId: String) async throws {
        if let addMediaCancelError { throw addMediaCancelError }
    }

    func addMediaTakeover(tripId: String, sessionId: String) async throws -> AddMediaTakeoverDTO {
        try addMediaTakeoverResult.get()
    }

    // MARK: - Stub data

    struct UpdateTripCall: Equatable {
        let id: String
        let name: String?
        let description: String?
        let category: String?
        let season: String?
        let privacyLevel: String?
        let coverUrl: String?
        let startDateUnix: Int?
        let endDateUnix: Int?
    }

    static let stubDesiredPlace = DesiredPlaceDTO(
        id: "dp-001", name: "Test Place", description: "Desc", imageUrl: nil, createdAt: 1_700_000_000
    )

    private static let stubTrip = TripDTO(
        id: "trip-001", name: "Test Trip", description: nil, category: nil, season: nil,
        coverUrl: nil, ownerUserId: "user-001", privacyLevel: "public", status: "published",
        isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0, mediaCount: 12,
        startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_000_000
    )

    private static let stubTripPins: [TripPinDTO] = [
        TripPinDTO(
            id: "pin-1",
            tripId: "trip-001",
            name: "Pin 1",
            description: nil,
            category: "vacation",
            latitude: 55.75,
            longitude: 37.62,
            startTimeUnix: nil,
            endTimeUnix: nil,
            tags: ["travel", "city"],
            privacyLevel: "public",
            media: (1...6).map { index in
                TripPinMediaDTO(
                    mediaId: "review-media-\(index)",
                    url: "https://example.com/review-\(index).jpg",
                    mediaType: "image",
                    privacyLevel: "public",
                    capturedAtUnix: nil
                )
            }
        ),
        TripPinDTO(
            id: "pin-2",
            tripId: "trip-001",
            name: "Pin 2",
            description: nil,
            category: "vacation",
            latitude: 55.76,
            longitude: 37.64,
            startTimeUnix: nil,
            endTimeUnix: nil,
            tags: ["trip"],
            privacyLevel: "public",
            media: (7...12).map { index in
                TripPinMediaDTO(
                    mediaId: "review-media-\(index)",
                    url: "https://example.com/review-\(index).jpg",
                    mediaType: "image",
                    privacyLevel: "public",
                    capturedAtUnix: nil
                )
            }
        )
    ]

    private static func stubBattleMedia() -> [StartBattleMediaDTO] {
        (1...8).map { index in
            StartBattleMediaDTO(
                photoBattleMediaId: "battle-media-\(index)",
                mediaType: "photo",
                url: "https://example.com/battle-\(index).jpg"
            )
        }
    }

    private static let stubRecommendedTrip = TripDTO(
        id: "trip-rec-001",
        name: "Рекомендованный маршрут",
        description: "Собранный маршрут по выбранной локации с акцентом на лучшие точки.",
        category: "vacation",
        season: "spring",
        coverUrl: nil,
        ownerUserId: "user-rec-001",
        privacyLevel: "public",
        status: "published",
        isPublished: true,
        isGenerated: true,
        likesCount: 73,
        dislikesCount: 1,
        participantsCount: 2,
        mediaCount: 4,
        startDateUnix: 1_700_000_000,
        endDateUnix: 1_700_020_000,
        createdAtUnix: 1_699_990_000,
        updatedAtUnix: 1_699_990_000
    )

    private static let stubRecommendationPins: [RecommendedPinDTO] = [
        RecommendedPinDTO(
            id: "rec-pin-001",
            tripId: "trip-rec-001",
            name: "Тайная улица",
            description: "Лучшее место для фото на закате и спокойных прогулок.",
            category: "vacation",
            latitude: 39.9042,
            longitude: 116.4074,
            locationName: "Пекин",
            mediaCount: 2,
            media: [
                FeedMediaDTO(
                    mediaId: "rec-pin-media-001",
                    url: "https://example.com/rec-pin-001.jpg",
                    mediaType: "photo"
                ),
                FeedMediaDTO(
                    mediaId: "rec-pin-media-002",
                    url: "https://example.com/rec-pin-002.jpg",
                    mediaType: "photo"
                )
            ]
        ),
        RecommendedPinDTO(
            id: "rec-pin-002",
            tripId: "trip-rec-001",
            name: "Скрытый дворик",
            description: "Уютный локальный уголок для короткой остановки и кофе.",
            category: "vacation",
            latitude: 39.9052,
            longitude: 116.4082,
            locationName: "Пекин",
            mediaCount: 1,
            media: [
                FeedMediaDTO(
                    mediaId: "rec-pin-media-003",
                    url: "https://example.com/rec-pin-003.jpg",
                    mediaType: "photo"
                )
            ]
        )
    ]

    private static let stubRecommendationMap = RecommendedMapDTO(
        media: [
            FeedMediaDTO(
                mediaId: "rec-media-001",
                url: "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg",
                mediaType: "photo"
            )
        ],
        pins: stubRecommendationPins,
        regionName: "Пекин",
        regionType: "city",
        snapshotToken: "stub-snapshot-token-001",
        trip: stubRecommendedTrip
    )
    private static let stubGetRecommendationsResponse = GetRecommendationsResponseDTO(map: stubRecommendationMap)
    private static let stubSaveRecommendationResponse = SaveRecommendationResponseDTO(trip: stubRecommendedTrip)

    private static let stubTripResponse = GetTripResponseDTO(trip: stubTrip, pins: stubTripPins, participants: [])

    private static let stubFeed: [FeedItemDTO] = [
        FeedItemDTO(
            trip: TripDTO(
                id: "trip-feed-001",
                name: "Парижская романтика (mock)",
                description: "Список медиа задан у каждого пина",
                category: "vacation",
                season: "spring",
                coverUrl: nil,
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 42,
                dislikesCount: 2,
                participantsCount: 12,
                mediaCount: 6,
                startDateUnix: 1_700_000_000,
                endDateUnix: 1_700_020_000,
                createdAtUnix: 1_699_900_000,
                updatedAtUnix: 1_699_900_000
            ),
            pins: [
                FeedPinDTO(
                    id: "pin-feed-001",
                    latitude: 48.8584,
                    longitude: 2.2945,
                    media: [
                        FeedMediaDTO(
                            mediaId: "mock-feed-001-001",
                            url: "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg",
                            mediaType: "photo"
                        ),
                        FeedMediaDTO(
                            mediaId: "mock-feed-001-002",
                            url: "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg",
                            mediaType: "photo"
                        )
                    ]
                ),
                FeedPinDTO(
                    id: "pin-feed-002",
                    latitude: 48.8606,
                    longitude: 2.3352,
                    media: [
                        FeedMediaDTO(
                            mediaId: "mock-feed-002-001",
                            url: "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg",
                            mediaType: "photo"
                        ),
                        FeedMediaDTO(
                            mediaId: "mock-feed-002-002",
                            url: "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg",
                            mediaType: "photo"
                        )
                    ]
                )
            ],
            media: []
        ),
        FeedItemDTO(
            trip: TripDTO(
                id: "trip-feed-002",
                name: "Горнолыжный тур в Альпы (mock)",
                description: "Каждый pin содержит минимум по одному медиа-объекту",
                category: "active",
                season: "winter",
                coverUrl: nil,
                ownerUserId: "user-002",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 38,
                dislikesCount: 1,
                participantsCount: 8,
                mediaCount: 4,
                startDateUnix: 1_698_000_000,
                endDateUnix: 1_698_400_000,
                createdAtUnix: 1_697_900_000,
                updatedAtUnix: 1_697_950_000
            ),
            pins: [
                FeedPinDTO(
                    id: "pin-feed-003",
                    latitude: 46.8182,
                    longitude: 8.2275,
                    media: [
                        FeedMediaDTO(
                            mediaId: "mock-feed-003-001",
                            url: "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg",
                            mediaType: "photo"
                        )
                    ]
                )
            ],
            media: []
        )
    ]
}
// swiftlint:enable file_length
