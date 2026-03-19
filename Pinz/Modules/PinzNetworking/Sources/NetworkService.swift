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

    // Feed
    func getFeed(limit: Int?, offset: Int?, category: String?, season: String?, locationId: Int?, locationName: String?, sortBy: String?) async throws -> [TripDTO]

    // Trips CRUD
    func getTrips() async throws -> [TripDTO]
    func getTrip(id: String) async throws -> TripDTO
    func updateTrip(id: String, name: String?, description: String?, category: String?, season: String?, privacyLevel: String?, coverUrl: String?, startDateUnix: Int?, endDateUnix: Int?) async throws -> TripDTO
    func deleteTrip(id: String) async throws

    // Trip actions
    func joinTripByToken(token: String) async throws -> JoinTripByTokenDTO
    func generateInviteLink(tripId: String, expiresInSeconds: Int?) async throws -> GenerateInviteLinkDTO
    func leaveTrip(id: String) async throws -> LeaveTripDTO
    func removeParticipant(tripId: String, userId: String) async throws
    func publishTrip(id: String, publishWhole: Bool, pinIds: [String]) async throws -> TripDTO
    func updateTripSettings(id: String, notificationsEnabled: Bool) async throws -> SuccessDTO
    func transferAdmin(id: String, newAdminUserId: String) async throws -> SuccessDTO
    func likeTrip(id: String) async throws -> SuccessDTO
    func dislikeTrip(id: String) async throws -> SuccessDTO
    func addTripToFavourites(id: String) async throws -> SuccessDTO
    func removeTripFromFavourites(id: String) async throws

    // Add-media flow
    func addMediaStart(tripId: String, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO
    func addMediaProcessGrouping(tripId: String, sessionId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO
    func addMediaApplyGroupsAndProcess(tripId: String, sessionId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO

    // Trip creation flow
    func createTrip(name: String, description: String?, category: String?, season: String?, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO
    func processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO
    func applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO
    func getTripReview(tripId: String) async throws -> GetTripReviewDTO
    func finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String]) async throws -> FinalizeTripDTO
}

// MARK: - Implementation

public final class NetworkService: NetworkServiceProtocol {
    private let provider = NetworkProvider<PinzAPI>()

    public init() {}

    // MARK: Auth

    public func devLogin(email: String) async throws -> UserTokensDTO {
        try await provider.request(.devLogin(email: email), type: UserTokensDTO.self)
    }

    public func submitEmail(email: String) async throws -> SubmitEmailDTO {
        try await provider.request(.submitEmail(email: email), type: SubmitEmailDTO.self)
    }

    public func verifyEmail(registrationId: String, verificationCode: String) async throws -> SuccessDTO {
        try await provider.request(.verifyEmail(registrationId: registrationId, verificationCode: verificationCode), type: SuccessDTO.self)
    }

    public func passkeyLoginBegin(email: String) async throws -> PasskeyOptionsDTO {
        try await provider.request(.passkeyLoginBegin(email: email), type: PasskeyOptionsDTO.self)
    }

    public func passkeyLoginFinish(email: String, credentialJSON: String) async throws -> UserTokensDTO {
        try await provider.request(.passkeyLoginFinish(email: email, credentialJSON: credentialJSON), type: UserTokensDTO.self)
    }

    public func passkeyRegisterBegin(registrationId: String, username: String) async throws -> PasskeyOptionsDTO {
        try await provider.request(.passkeyRegisterBegin(registrationId: registrationId, username: username), type: PasskeyOptionsDTO.self)
    }

    public func passkeyRegisterFinish(registrationId: String, credentialJSON: String) async throws -> UserTokensDTO {
        try await provider.request(.passkeyRegisterFinish(registrationId: registrationId, credentialJSON: credentialJSON), type: UserTokensDTO.self)
    }

    public func refreshToken(refreshToken: String) async throws -> RefreshTokenDTO {
        try await provider.request(.refreshToken(refreshToken: refreshToken), type: RefreshTokenDTO.self)
    }

    public func logout(refreshToken: String) async throws -> SuccessDTO {
        try await provider.request(.logout(refreshToken: refreshToken), type: SuccessDTO.self)
    }

    // MARK: Feed

    public func getFeed(limit: Int? = nil, offset: Int? = nil, category: String? = nil, season: String? = nil, locationId: Int? = nil, locationName: String? = nil, sortBy: String? = nil) async throws -> [TripDTO] {
        try await provider.request(.getFeed(limit: limit, offset: offset, category: category, season: season, locationId: locationId, locationName: locationName, sortBy: sortBy), type: [TripDTO].self)
    }

    // MARK: Trips CRUD

    public func getTrips() async throws -> [TripDTO] {
        try await provider.request(.getTrips, type: [TripDTO].self)
    }

    public func getTrip(id: String) async throws -> TripDTO {
        try await provider.request(.getTrip(id: id), type: TripDTO.self)
    }

    public func updateTrip(id: String, name: String? = nil, description: String? = nil, category: String? = nil, season: String? = nil, privacyLevel: String? = nil, coverUrl: String? = nil, startDateUnix: Int? = nil, endDateUnix: Int? = nil) async throws -> TripDTO {
        try await provider.request(.updateTrip(id: id, name: name, description: description, category: category, season: season, privacyLevel: privacyLevel, coverUrl: coverUrl, startDateUnix: startDateUnix, endDateUnix: endDateUnix), type: TripDTO.self)
    }

    public func deleteTrip(id: String) async throws {
        _ = try await provider.requestRaw(.deleteTrip(id: id))
    }

    // MARK: Trip actions

    public func joinTripByToken(token: String) async throws -> JoinTripByTokenDTO {
        try await provider.request(.joinTripByToken(token: token), type: JoinTripByTokenDTO.self)
    }

    public func generateInviteLink(tripId: String, expiresInSeconds: Int? = nil) async throws -> GenerateInviteLinkDTO {
        try await provider.request(.generateInviteLink(tripId: tripId, expiresInSeconds: expiresInSeconds), type: GenerateInviteLinkDTO.self)
    }

    public func leaveTrip(id: String) async throws -> LeaveTripDTO {
        try await provider.request(.leaveTrip(id: id), type: LeaveTripDTO.self)
    }

    public func removeParticipant(tripId: String, userId: String) async throws {
        _ = try await provider.requestRaw(.removeParticipant(tripId: tripId, userId: userId))
    }

    public func publishTrip(id: String, publishWhole: Bool, pinIds: [String]) async throws -> TripDTO {
        try await provider.request(.publishTrip(id: id, publishWhole: publishWhole, pinIds: pinIds), type: TripDTO.self)
    }

    public func updateTripSettings(id: String, notificationsEnabled: Bool) async throws -> SuccessDTO {
        try await provider.request(.updateTripSettings(id: id, notificationsEnabled: notificationsEnabled), type: SuccessDTO.self)
    }

    public func transferAdmin(id: String, newAdminUserId: String) async throws -> SuccessDTO {
        try await provider.request(.transferAdmin(id: id, newAdminUserId: newAdminUserId), type: SuccessDTO.self)
    }

    public func likeTrip(id: String) async throws -> SuccessDTO {
        try await provider.request(.likeTrip(id: id), type: SuccessDTO.self)
    }

    public func dislikeTrip(id: String) async throws -> SuccessDTO {
        try await provider.request(.dislikeTrip(id: id), type: SuccessDTO.self)
    }

    public func addTripToFavourites(id: String) async throws -> SuccessDTO {
        try await provider.request(.addTripToFavourites(id: id), type: SuccessDTO.self)
    }

    public func removeTripFromFavourites(id: String) async throws {
        _ = try await provider.requestRaw(.removeTripFromFavourites(id: id))
    }

    // MARK: Add-media flow

    public func addMediaStart(tripId: String, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO {
        try await provider.request(.addMediaStart(tripId: tripId, filesToUpload: filesToUpload), type: CreateTripDTO.self)
    }

    public func addMediaProcessGrouping(tripId: String, sessionId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO {
        try await provider.request(.addMediaProcessGrouping(tripId: tripId, sessionId: sessionId, media: media), type: ProcessMediaGroupingDTO.self)
    }

    public func addMediaApplyGroupsAndProcess(tripId: String, sessionId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO {
        try await provider.request(.addMediaApplyGroupsAndProcess(tripId: tripId, sessionId: sessionId, draftPins: draftPins, deletedMediaIds: deletedMediaIds), type: ApplyGroupsAndProcessDTO.self)
    }

    // MARK: Trip creation flow

    public func createTrip(name: String, description: String? = nil, category: String? = nil, season: String? = nil, filesToUpload: [FileToUploadDTO]) async throws -> CreateTripDTO {
        try await provider.request(.createTrip(name: name, description: description, category: category, season: season, filesToUpload: filesToUpload), type: CreateTripDTO.self)
    }

    public func processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO]) async throws -> ProcessMediaGroupingDTO {
        try await provider.request(.processMediaGrouping(tripId: tripId, media: media), type: ProcessMediaGroupingDTO.self)
    }

    public func applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String]) async throws -> ApplyGroupsAndProcessDTO {
        try await provider.request(.applyGroupsAndProcess(tripId: tripId, draftPins: draftPins, deletedMediaIds: deletedMediaIds), type: ApplyGroupsAndProcessDTO.self)
    }

    public func getTripReview(tripId: String) async throws -> GetTripReviewDTO {
        try await provider.request(.getTripReview(tripId: tripId), type: GetTripReviewDTO.self)
    }

    public func finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String]) async throws -> FinalizeTripDTO {
        try await provider.request(.finalizeTrip(tripId: tripId, pinUpdates: pinUpdates, mediaToDelete: mediaToDelete), type: FinalizeTripDTO.self)
    }
}
// swiftlint:enable file_length function_parameter_count
