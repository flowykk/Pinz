// swiftlint:disable file_length function_parameter_count
import Moya
import Foundation
import PinzBase
import PinzDomain

// MARK: - Protocol

public protocol NetworkServiceProtocol {
    // Auth
    func devLogin(email: String) async throws -> UserTokensDTO
    func submitEmail(email: String) async throws -> SubmitEmailDTO
    func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessDTO
    func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsDTO
    func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensDTO
    func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsDTO
    func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensDTO
    func refreshToken(refreshToken: String) async throws -> RefreshTokenDTO
    func logout(refreshToken: String) async throws -> SuccessDTO

    // Profile
    func getProfile() async throws -> ProfileResponseDTO
    func updateProfile(username: String) async throws -> ProfileResponseDTO
    func deleteAvatar() async throws -> ProfileResponseDTO
    func deleteAccount() async throws -> DeleteAccountResponseDTO
    func requestAvatarUpload(filename: String, contentType: String) async throws -> AvatarUploadResponseDTO
    func confirmAvatarUpload(s3Key: String) async throws -> ProfileResponseDTO
    func changeEmail(userId: String?, newEmail: String) async throws -> ChangeEmailResponseDTO
    func confirmEmailChange(verificationCode: String) async throws -> ProfileResponseDTO

    // Feed
    func getFeed(
        limit: Int?,
        offset: Int?,
        category: String?,
        season: String?,
        locationId: Int?,
        sortBy: String?
    ) async throws -> [TripDTO]

    // Trips CRUD
    func getTrips() async throws -> [TripDTO]
    func getFavouriteTrips(limit: Int?, offset: Int?) async throws -> [TripDTO]
    func getTrip(id: String) async throws -> GetTripResponseDTO
    func requestTripCoverUpload(id: String, filename: String, contentType: String) async throws -> TripCoverUploadResponseDTO
    func confirmTripCoverUpload(id: String, s3Key: String) async throws -> TripDTO
    func updateTrip(
        id: String,
        name: String?,
        description: String?,
        category: String?,
        season: String?,
        privacyLevel: String?,
        coverUrl: String?,
        startDateUnix: Int?,
        endDateUnix: Int?
    ) async throws -> TripDTO
    func deleteTrip(id: String) async throws

    // Trip actions
    func joinTripByToken(token: String) async throws -> JoinTripByTokenDTO
    func generateInviteLink(tripId: String, expiresInSeconds: Int?) async throws -> GenerateInviteLinkDTO
    func leaveTrip(id: String) async throws -> LeaveTripDTO
    func removeParticipant(tripId: String, userId: String) async throws
    func publishTrip(id: String, publishWhole: Bool, pinIds: [String]) async throws -> TripDTO
    func updateTripSettings(id: String, notificationsEnabled: Bool) async throws -> SuccessDTO
    func likeTrip(id: String) async throws -> SuccessDTO
    func dislikeTrip(id: String) async throws -> SuccessDTO
    func addTripToFavourites(id: String) async throws -> SuccessDTO
    func removeTripFromFavourites(id: String) async throws

    // S3 upload
    func uploadToS3(url: String, data: Data, contentType: String) async throws

    // Trip creation flow
    func createTrip(
        name: String,
        description: String?,
        category: String?,
        season: String?,
        filesToUpload: [FileToUploadDTO]
    ) async throws -> CreateTripDTO
    func processMediaGrouping(
        tripId: String,
        media: [MediaMetaEntryDTO]
    ) async throws -> ProcessMediaGroupingDTO
    func applyGroupsAndProcess(
        tripId: String,
        draftPins: [DraftPinInputDTO],
        deletedMediaIds: [String]
    ) async throws -> ApplyGroupsAndProcessDTO
    func getTripReview(tripId: String) async throws -> GetTripReviewDTO
    func finalizeTrip(
        tripId: String,
        pinUpdates: [PinUpdateInputDTO],
        mediaToDelete: [String]
    ) async throws -> FinalizeTripDTO
}

// MARK: - Implementation

public final class NetworkService: NetworkServiceProtocol {
    public static let shared = NetworkService()

    private let provider: NetworkProvider<PinzAPI>

    public init() {
        let stub: Bool = false
        self.provider = NetworkProvider<PinzAPI>(stub: stub, stubDelay: 0.5)
    }

    // MARK: Auth

    public func devLogin(
        email: String
    ) async throws -> UserTokensDTO {
        try await provider.request(
            .devLogin(
                email: email
            ),
            type: UserTokensDTO.self
        )
    }
    
    public func submitEmail(
        email: String
    ) async throws -> SubmitEmailDTO {
        try await provider.request(
            .submitEmail(
                email: email
            ),
            type: SubmitEmailDTO.self
        )
    }
    
    public func verifyEmail(
        registrationId: String,
        verificationCode: String
    ) async throws -> SuccessDTO {
        try await provider.request(
            .verifyEmail(
                registrationId: registrationId,
                verificationCode: verificationCode
            ),
            type: SuccessDTO.self
        )
    }
    
    public func passkeyLoginBegin(
        email: String
    ) async throws -> PasskeyOptionsDTO {
        try await provider.request(
            .passkeyLoginBegin(
                email: email
            ),
            type: PasskeyOptionsDTO.self
        )
    }
    
    public func passkeyLoginFinish(
        email: String,
        credentialJSON: String
    ) async throws -> UserTokensDTO {
        try await provider.request(
            .passkeyLoginFinish(
                email: email,
                credentialJSON: credentialJSON
            ),
            type: UserTokensDTO.self
        )
    }
    
    public func passkeyRegisterBegin(
        registrationId: String,
        username: String
    ) async throws -> PasskeyOptionsDTO {
        try await provider.request(
            .passkeyRegisterBegin(
                registrationId: registrationId,
                username: username
            ),
            type: PasskeyOptionsDTO.self
        )
    }
    
    public func passkeyRegisterFinish(
        registrationId: String,
        credentialJSON: String
    ) async throws -> UserTokensDTO {
        try await provider.request(
            .passkeyRegisterFinish(
                registrationId: registrationId,
                credentialJSON: credentialJSON
            ),
            type: UserTokensDTO.self
        )
    }
    
    public func refreshToken(
        refreshToken: String
    ) async throws -> RefreshTokenDTO {
        try await provider.request(
            .refreshToken(
                refreshToken: refreshToken
            ),
            type: RefreshTokenDTO.self
        )
    }
    
    public func logout(
        refreshToken: String
    ) async throws -> SuccessDTO {
        try await provider.request(
            .logout(
                refreshToken: refreshToken
            ),
            type: SuccessDTO.self
        )
    }

    // MARK: Profile

    public func getProfile() async throws -> ProfileResponseDTO {
        try await provider.request(
            .getProfile,
            type: ProfileResponseDTO.self
        )
    }

    public func updateProfile(
        username: String
    ) async throws -> ProfileResponseDTO {
        try await provider.request(
            .updateProfile(
                username: username
            ),
            type: ProfileResponseDTO.self
        )
    }

    public func deleteAccount() async throws -> DeleteAccountResponseDTO {
        try await provider.request(
            .deleteAccount,
            type: DeleteAccountResponseDTO.self
        )
    }

    public func deleteAvatar() async throws -> ProfileResponseDTO {
        try await provider.request(
            .deleteAvatar,
            type: ProfileResponseDTO.self
        )
    }

    public func requestAvatarUpload(
        filename: String,
        contentType: String
    ) async throws -> AvatarUploadResponseDTO {
        try await provider.request(
            .requestAvatarUpload(
                filename: filename,
                contentType: contentType
            ),
            type: AvatarUploadResponseDTO.self
        )
    }

    public func confirmAvatarUpload(s3Key: String) async throws -> ProfileResponseDTO {
        try await provider.request(
            .confirmAvatarUpload(
                s3Key: s3Key
            ),
            type: ProfileResponseDTO.self
        )
    }

    public func requestTripCoverUpload(
        id: String,
        filename: String,
        contentType: String
    ) async throws -> TripCoverUploadResponseDTO {
        try await provider.request(
            .requestTripCoverUpload(
                id: id,
                filename: filename,
                contentType: contentType
            ),
            type: TripCoverUploadResponseDTO.self
        )
    }

    public func confirmTripCoverUpload(id: String, s3Key: String) async throws -> TripDTO {
        try await provider.request(
            .confirmTripCoverUpload(
                id: id,
                s3Key: s3Key
            ),
            type: TripDTO.self
        )
    }

    public func changeEmail(userId: String?, newEmail: String) async throws -> ChangeEmailResponseDTO {
        try await provider.request(
            .changeEmail(
                userId: userId,
                newEmail: newEmail
            ),
            type: ChangeEmailResponseDTO.self
        )
    }

    public func confirmEmailChange(verificationCode: String) async throws -> ProfileResponseDTO {
        try await provider.request(
            .confirmEmailChange(
                verificationCode: verificationCode
            ),
            type: ProfileResponseDTO.self
        )
    }

    // MARK: Feed

    public func getFeed(
        limit: Int? = nil,
        offset: Int? = nil,
        category: String? = nil,
        season: String? = nil,
        locationId: Int? = nil,
        sortBy: String? = nil
    ) async throws -> [TripDTO] {
        try await provider.request(
            .getFeed(
                limit: limit,
                offset: offset,
                category: category,
                season: season,
                locationId: locationId,
                sortBy: sortBy
            ),
            type: [TripDTO].self
        )
    }
    
    // MARK: Trips CRUD
    
    public func getTrips() async throws -> [TripDTO] {
        try await provider.request(
            .getTrips,
            type: [TripDTO].self
        )
    }

    public func getFavouriteTrips(
        limit: Int? = nil,
        offset: Int? = nil
    ) async throws -> [TripDTO] {
        try await retryOnUnauthorized { [self] in
            try await provider.request(
                .getFavouriteTrips(limit: limit, offset: offset),
                type: [TripDTO].self
            )
        }
    }

    public func getTrip(
        id: String
    ) async throws -> GetTripResponseDTO {
        try await provider.request(
            .getTrip(
                id: id
            ),
            type: GetTripResponseDTO.self
        )
    }

    public func updateTrip(
        id: String,
        name: String? = nil,
        description: String? = nil,
        category: String? = nil,
        season: String? = nil,
        privacyLevel: String? = nil,
        coverUrl: String? = nil,
        startDateUnix: Int? = nil,
        endDateUnix: Int? = nil
    ) async throws -> TripDTO {
        try await provider.request(
            .updateTrip(
                id: id,
                name: name,
                description: description,
                category: category,
                season: season,
                privacyLevel: privacyLevel,
                coverUrl: coverUrl,
                startDateUnix: startDateUnix,
                endDateUnix: endDateUnix
            ),
            type: TripDTO.self
        )
    }

    public func deleteTrip(
        id: String
    ) async throws {
        _ = try await provider.requestRaw(
            .deleteTrip(
                id: id
            )
        )
    }

    // MARK: Trip actions

    public func joinTripByToken(
        token: String
    ) async throws -> JoinTripByTokenDTO {
        try await provider.request(
            .joinTripByToken(
                token: token
            ),
            type: JoinTripByTokenDTO.self
        )
    }

    public func generateInviteLink(
        tripId: String,
        expiresInSeconds: Int? = nil
    ) async throws -> GenerateInviteLinkDTO {
        try await provider.request(
            .generateInviteLink(
                tripId: tripId,
                expiresInSeconds: expiresInSeconds
            ),
            type: GenerateInviteLinkDTO.self
        )
    }

    public func leaveTrip(
        id: String
    ) async throws -> LeaveTripDTO {
        try await provider.request(
            .leaveTrip(
                id: id
            ),
            type: LeaveTripDTO.self
        )
    }
    
    public func removeParticipant(
        tripId: String,
        userId: String
    ) async throws {
        _ = try await provider.requestRaw(
            .removeParticipant(
                tripId: tripId,
                userId: userId
            )
        )
    }

    public func publishTrip(
        id: String,
        publishWhole: Bool,
        pinIds: [String]
    ) async throws -> TripDTO {
        try await provider.request(
            .publishTrip(
                id: id,
                publishWhole: publishWhole,
                pinIds: pinIds
            ),
            type: TripDTO.self
        )
    }

    public func updateTripSettings(
        id: String,
        notificationsEnabled: Bool
    ) async throws -> SuccessDTO {
        try await provider.request(
            .updateTripSettings(
                id: id,
                notificationsEnabled: notificationsEnabled
            ),
            type: SuccessDTO.self
        )
    }

    public func likeTrip(
        id: String
    ) async throws -> SuccessDTO {
        try await provider.request(
            .likeTrip(
                id: id
            ),
            type: SuccessDTO.self
        )
    }
    
    public func dislikeTrip(
        id: String
    ) async throws -> SuccessDTO {
        try await provider.request(
            .dislikeTrip(
                id: id
            ),
            type: SuccessDTO.self
        )
    }
    
    public func addTripToFavourites(
        id: String
    ) async throws -> SuccessDTO {
        try await provider.request(
            .addTripToFavourites(
                id: id
            ),
            type: SuccessDTO.self
        )
    }
    
    public func removeTripFromFavourites(
        id: String
    ) async throws {
        _ = try await provider.requestRaw(
            .removeTripFromFavourites(
                id: id
            )
        )
    }

    // MARK: Retry on Unauthorized

    private func retryOnUnauthorized<T>(
        _ perform: @escaping () async throws -> T
    ) async throws -> T {
        do {
            return try await perform()
        } catch let httpError as HTTPError where httpError == .unauthorized {
            guard let storedRefreshToken = TokenStorage.shared.refreshToken else { throw httpError }
            let newAccessToken = try await refreshToken(refreshToken: storedRefreshToken).accessToken
            TokenStorage.shared.save(accessToken: newAccessToken, refreshToken: storedRefreshToken)
            return try await perform()
        }
    }

    // MARK: S3 Upload

    public func uploadToS3(url: String, data: Data, contentType: String) async throws {
        guard let uploadURL = URL(string: url) else { throw URLError(.badURL) }
        var request = URLRequest(url: uploadURL)
        request.httpMethod = "PUT"
        request.setValue(contentType, forHTTPHeaderField: "Content-Type")
        let (_, response) = try await URLSession.shared.upload(for: request, from: data)
        guard let http = response as? HTTPURLResponse else {
            throw URLError(.badServerResponse)
        }
        guard (200...299).contains(http.statusCode) else {
            #if DEBUG
            print("S3 upload failed: HTTP \(http.statusCode) for \(url)")
            #endif
            throw NSError(
                domain: "PinzS3Upload",
                code: http.statusCode,
                userInfo: [NSLocalizedDescriptionKey: "S3 upload failed (HTTP \(http.statusCode))"]
            )
        }
    }

    // MARK: Trip creation flow

    public func createTrip(
        name: String,
        description: String? = nil,
        category: String? = nil,
        season: String? = nil,
        filesToUpload: [FileToUploadDTO]
    ) async throws -> CreateTripDTO {
        try await retryOnUnauthorized { [self] in
            try await provider.request(
                .createTrip(
                    name: name,
                    description: description,
                    category: category,
                    season: season,
                    filesToUpload: filesToUpload
                ),
                type: CreateTripDTO.self
            )
        }
    }

    public func processMediaGrouping(
        tripId: String,
        media: [MediaMetaEntryDTO]
    ) async throws -> ProcessMediaGroupingDTO {
        try await retryOnUnauthorized { [self] in
            try await provider.request(
                .processMediaGrouping(
                    tripId: tripId,
                    media: media
                ),
                type: ProcessMediaGroupingDTO.self
            )
        }
    }

    @discardableResult
    public func applyGroupsAndProcess(
        tripId: String,
        draftPins: [DraftPinInputDTO],
        deletedMediaIds: [String]
    ) async throws -> ApplyGroupsAndProcessDTO {
        try await provider.request(
            .applyGroupsAndProcess(
                tripId: tripId,
                draftPins: draftPins,
                deletedMediaIds: deletedMediaIds
            ),
            type: ApplyGroupsAndProcessDTO.self
        )
    }
    
    public func getTripReview(
        tripId: String
    ) async throws -> GetTripReviewDTO {
        try await retryOnUnauthorized { [self] in
            try await provider.request(.getTripReview(tripId: tripId), type: GetTripReviewDTO.self)
        }
    }
    
    public func finalizeTrip(
        tripId: String,
        pinUpdates: [PinUpdateInputDTO],
        mediaToDelete: [String]
    ) async throws -> FinalizeTripDTO {
        try await provider.request(
            .finalizeTrip(
                tripId: tripId,
                pinUpdates: pinUpdates,
                mediaToDelete: mediaToDelete
            ),
            type: FinalizeTripDTO.self
        )
    }
}
// swiftlint:enable file_length function_parameter_count
