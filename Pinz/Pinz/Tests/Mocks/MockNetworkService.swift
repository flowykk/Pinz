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

    var getFeedResult: Result<[TripDTO], Error> = .success([])
    var getTripsResult: Result<[TripDTO], Error> = .success([])
    var getTripResult: Result<GetTripResponseDTO, Error> = .success(MockNetworkService.stubTripResponse)
    var updateTripResult: Result<TripDTO, Error> = .success(MockNetworkService.stubTrip)
    var updateTripCall: UpdateTripCall?
    var deleteTripError: Error?

    var joinTripByTokenResult: Result<JoinTripByTokenDTO, Error> = .success(JoinTripByTokenDTO(tripId: "trip-001", alreadyJoined: false))
    var generateInviteLinkResult: Result<GenerateInviteLinkDTO, Error> = .success(
        GenerateInviteLinkDTO(inviteLinkId: "link-001", inviteUrl: "https://pinz.website/join/token", token: "token", expiresAtUnix: nil)
    )
    var leaveTripResult: Result<LeaveTripDTO, Error> = .success(LeaveTripDTO(success: true, tripDeleted: false))
    var removeParticipantError: Error?
    var publishTripResult: Result<TripDTO, Error> = .success(MockNetworkService.stubTrip)
    var updateTripSettingsResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var updateTripSettingsCall: (id: String, notificationsEnabled: Bool)?
    var likeTripResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var dislikeTripResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var addTripToFavouritesResult: Result<SuccessDTO, Error> = .success(SuccessDTO(success: true))
    var getFavouriteTripsResult: Result<[TripDTO], Error> = .success([])
    var removeTripFromFavouritesError: Error?

    // MARK: - Profile

    var getProfileResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var updateProfileResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var deleteAccountResult: Result<DeleteAccountResponseDTO, Error> = .success(DeleteAccountResponseDTO(success: true))
    var requestAvatarUploadResult: Result<AvatarUploadResponseDTO, Error> = .success(
        AvatarUploadResponseDTO(uploadUrl: "https://example.com/upload", s3Key: "mock-s3-key")
    )
    var confirmAvatarUploadResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var changeEmailResult: Result<ChangeEmailResponseDTO, Error> = .success(ChangeEmailResponseDTO(success: true))
    var confirmEmailChangeResult: Result<ProfileResponseDTO, Error> = .success(ProfileResponseDTO(nickname: "tester", email: "test@example.com"))
    var requestAvatarUploadCall: (filename: String, contentType: String)?
    var confirmAvatarUploadCall: String?
    var uploadToS3Call: (url: String, dataBytes: Int, contentType: String)?

    // MARK: - Trip creation / add-media

    var createTripResult: Result<CreateTripDTO, Error> = .success(CreateTripDTO(tripId: "trip-001", status: "created", uploadUrls: []))
    var processMediaGroupingResult: Result<ProcessMediaGroupingDTO, Error> = .success(
        ProcessMediaGroupingDTO(tripId: "trip-001", status: "processed", draftPins: [])
    )
    var applyGroupsAndProcessResult: Result<ApplyGroupsAndProcessDTO, Error> = .success(
        ApplyGroupsAndProcessDTO(message: "processing", status: "ok")
    )
    var getTripReviewResult: Result<GetTripReviewDTO, Error> = .success(
        GetTripReviewDTO(tripId: "trip-001", status: "ready", pins: [], similar: [])
    )
    var finalizeTripResult: Result<FinalizeTripDTO, Error> = .success(
        FinalizeTripDTO(tripId: "trip-001", status: "finalized", message: "done")
    )
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

    func getFeed(limit: Int?, offset: Int?, category: String?, season: String?, locationId: Int?, sortBy: String?) async throws -> [TripDTO] {
        try getFeedResult.get()
    }

    // MARK: - Trips CRUD

    func getTrips() async throws -> [TripDTO] { try getTripsResult.get() }
    func getTrip(id: String) async throws -> GetTripResponseDTO { try getTripResult.get() }
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
        try updateTripResult.get()
    }
    func deleteTrip(id: String) async throws {
        if let error = deleteTripError { throw error }
    }

    // MARK: - Trip actions

    func joinTripByToken(token: String) async throws -> JoinTripByTokenDTO { try joinTripByTokenResult.get() }
    func generateInviteLink(tripId: String, expiresInSeconds: Int?) async throws -> GenerateInviteLinkDTO { try generateInviteLinkResult.get() }
    func leaveTrip(id: String) async throws -> LeaveTripDTO { try leaveTripResult.get() }
    func removeParticipant(tripId: String, userId: String) async throws {
        if let error = removeParticipantError { throw error }
    }
    func publishTrip(id: String, publishWhole: Bool, pinIds: [String]) async throws -> TripDTO { try publishTripResult.get() }
    func updateTripSettings(id: String, notificationsEnabled: Bool) async throws -> SuccessDTO {
        updateTripSettingsCall = (id: id, notificationsEnabled: notificationsEnabled)
        return try updateTripSettingsResult.get()
    }
    func likeTrip(id: String) async throws -> SuccessDTO { try likeTripResult.get() }
    func dislikeTrip(id: String) async throws -> SuccessDTO { try dislikeTripResult.get() }
    func addTripToFavourites(id: String) async throws -> SuccessDTO { try addTripToFavouritesResult.get() }
    func removeTripFromFavourites(id: String) async throws {
        if let error = removeTripFromFavouritesError { throw error }
    }
    func getProfile() async throws -> ProfileResponseDTO { try getProfileResult.get() }
    func updateProfile(username: String) async throws -> ProfileResponseDTO {
        try updateProfileResult.get()
    }
    func deleteAccount() async throws -> DeleteAccountResponseDTO { try deleteAccountResult.get() }
    func requestAvatarUpload(filename: String, contentType: String) async throws -> AvatarUploadResponseDTO {
        requestAvatarUploadCall = (filename, contentType)
        return try requestAvatarUploadResult.get()
    }
    func confirmAvatarUpload(s3Key: String) async throws -> ProfileResponseDTO {
        confirmAvatarUploadCall = s3Key
        return try confirmAvatarUploadResult.get()
    }
    func changeEmail(userId: String?, newEmail: String) async throws -> ChangeEmailResponseDTO {
        try changeEmailResult.get()
    }
    func confirmEmailChange(verificationCode: String) async throws -> ProfileResponseDTO { try confirmEmailChangeResult.get() }
    func getFavouriteTrips(limit: Int?, offset: Int?) async throws -> [TripDTO] {
        try getFavouriteTripsResult.get()
    }

    func uploadToS3(url: String, data: Data, contentType: String) async throws {
        uploadToS3Call = (url, data.count, contentType)
        if let uploadToS3Error { throw uploadToS3Error }
    }

    // MARK: - Trip creation flow

    func createTrip(name: String, description: String?, category: String?, season: String?, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO { try createTripResult.get() }
    func processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO { try processMediaGroupingResult.get() }
    func applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO { try applyGroupsAndProcessResult.get() }
    func getTripReview(tripId: String) async throws -> GetTripReviewDTO { try getTripReviewResult.get() }
    func finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String]) async throws -> FinalizeTripDTO { try finalizeTripResult.get() }

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

    private static let stubTrip = TripDTO(
        id: "trip-001", name: "Test Trip", description: nil, category: nil, season: nil,
        coverUrl: nil, ownerUserId: "user-001", privacyLevel: "public", status: "published",
        isPublished: true, isGenerated: false, likesCount: 0, dislikesCount: 0,
        startDateUnix: nil, endDateUnix: nil, createdAtUnix: 1_700_000_000, updatedAtUnix: 1_700_000_000
    )
    private static let stubTripResponse = GetTripResponseDTO(trip: stubTrip, pins: [])
}
// swiftlint:enable file_length
