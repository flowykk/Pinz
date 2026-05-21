// swiftlint:disable file_length
import Moya
import Foundation
import PinzBase
import PinzDomain

enum PinzAPI {
    // Auth
    case devLogin(email: String)
    case submitEmail(email: String)
    case verifyEmail(registrationId: String, verificationCode: String)
    case passkeyLoginBegin(email: String)
    case passkeyLoginFinish(email: String, credentialJSON: String)
    case passkeyRegisterBegin(registrationId: String, username: String)
    case passkeyRegisterFinish(registrationId: String, credentialJSON: String)
    case refreshToken(refreshToken: String)
    case logout(refreshToken: String)

    // Profile
    case getProfile
    case getProfileStats
    case getPublicUserProfile(id: String)
    case getVisitedLocations(type: String?)
    case deleteAccount
    case deleteAvatar
    case updateProfile(username: String)
    case requestAvatarUpload(filename: String, contentType: String)
    case confirmAvatarUpload(s3Key: String)
    case changeEmail(userId: String?, newEmail: String)
    case confirmEmailChange(verificationCode: String)
    case registerDeviceToken(apnsToken: String)
    case unregisterDeviceToken(apnsToken: String)

    // Feed
    case getFeed(limit: Int?, offset: Int?, category: String?, season: String?, city: String?, country: String?, sortBy: String?)
    case getRecommendations(city: String?, country: String?, category: String?, season: String?)
    case saveRecommendation(snapshotToken: String, pinIds: [String], city: String?, country: String?, category: String?, season: String?)

    // Trips CRUD
    case getTrips
    case getFavouriteTrips(limit: Int?, offset: Int?)
    case getTrip(id: String)
    case requestTripCoverUpload(id: String, filename: String, contentType: String)
    case confirmTripCoverUpload(id: String, s3Key: String)
    case updateTrip(
        id: String, name: String?, description: String?, category: String?,
        season: String?, privacyLevel: String?, coverUrl: String?,
        startDateUnix: Int?, endDateUnix: Int?
    )
    case deleteTrip(id: String)

    // Trip actions
    case joinTripByToken(token: String)
    case generateInviteLink(tripId: String, expiresInSeconds: Int?)
    case leaveTrip(id: String)
    case removeParticipant(tripId: String, userId: String)
    case publishTrip(id: String, publishWhole: Bool, pinIds: [String])
    case updateTripSettings(id: String, notificationsEnabled: Bool)
    case likeTrip(id: String)
    case dislikeTrip(id: String)
    case addTripToFavourites(id: String)
    case removeTripFromFavourites(id: String)

    // Trip creation flow
    case createTrip(name: String, description: String?, category: String?, season: String?, filesToUpload: [FileToUploadDTO])
    case processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO])
    case applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String])
    case getTripReview(tripId: String)
    case finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String])

    // Pin upload flow
    case pinUploadStart(tripId: String, targetPinId: String?, filesToUpload: [FileToUploadDTO])
    case pinUploadRequestUploadUrls(tripId: String, sessionId: String, filesToUpload: [FileToUploadDTO])
    case pinUploadCommitUpload(
        tripId: String, sessionId: String, s3Key: String, mediaType: String,
        capturedAtUnix: Int?, latitude: Double?, longitude: Double?
    )
    case pinUploadProcess(tripId: String, sessionId: String)
    case pinUploadGetReview(tripId: String, sessionId: String)
    case pinUploadFinalize(tripId: String, sessionId: String, input: PinUploadFinalizeInputDTO)
    case pinUploadCancel(tripId: String, sessionId: String)

    // Trip add-media flow
    case addMediaStart(tripId: String, filesToUpload: [FileToUploadDTO])
    case addMediaRequestUploadUrls(tripId: String, sessionId: String, filesToUpload: [FileToUploadDTO])
    case addMediaCommitUpload(tripId: String, sessionId: String, s3Key: String, mediaType: String, capturedAt: String?, latitude: Double?, longitude: Double?)
    case addMediaGetSessionMedia(tripId: String, sessionId: String)
    case addMediaProcessGrouping(tripId: String, sessionId: String, addMore: Bool)
    case addMediaGetGrouping(tripId: String, sessionId: String)
    case addMediaApplyGroupsAndProcess(tripId: String, sessionId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String])
    case addMediaGetReview(tripId: String, sessionId: String)
    case addMediaConfirm(tripId: String, sessionId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String])
    case addMediaCancel(tripId: String, sessionId: String)
    case addMediaTakeover(tripId: String, sessionId: String)

    // Photo battles
    case startBattle(tripId: String)
    case submitBattleResult(tripId: String, battleId: String, winnerMediaId: String)

    // Privacy
    case setTripPrivacy(tripId: String, privacyLevel: String)
    case setPinPrivacy(tripId: String, pinId: String, privacyLevel: String)
    case setMediaPrivacy(tripId: String, mediaId: String, privacyLevel: String)

    // Desired places
    case getDesiredPlaces
    case createDesiredPlace(name: String, description: String, s3Key: String?)
    case requestDesiredPlaceImageUpload(filename: String, contentType: String)
    case updateDesiredPlace(placeId: String, name: String, description: String, imageS3Key: String?)
    case deleteDesiredPlace(placeId: String)
    case deleteDesiredPlaceImage(placeId: String)

    // Pins CRUD
    case getPin(tripId: String, pinId: String)
    case deletePin(tripId: String, pinId: String)
    case updatePin(
        tripId: String, pinId: String,
        name: String?, description: String?, category: String?,
        latitude: Double?, longitude: Double?,
        startTimeUnix: Int?, endTimeUnix: Int?,
        tags: [String]?, tagsSet: Bool?
    )
    case deletePinMedia(tripId: String, pinId: String, mediaId: String)
    case searchPins(q: String, limit: Int?, offset: Int?)
}

// MARK: - TargetType

extension PinzAPI: TargetType {
    var baseURL: URL {
        if let url = URL(string: PinzLaunchArgs.baseURL) {
            return url
        }
        return URL(string: "https://pinz.website")!
    }

    var path: String {
        let endpointPath: String
        switch self {
        case .devLogin: endpointPath = "/auth/dev-login"
        case .submitEmail: endpointPath = "/auth/email"
        case .verifyEmail: endpointPath = "/auth/verify-email"
        case .passkeyLoginBegin: endpointPath = "/auth/passkey/login/begin"
        case .passkeyLoginFinish: endpointPath = "/auth/passkey/login/finish"
        case .passkeyRegisterBegin: endpointPath = "/auth/passkey/register/begin"
        case .passkeyRegisterFinish: endpointPath = "/auth/passkey/register/finish"
        case .refreshToken: endpointPath = "/auth/refresh"
        case .logout: endpointPath = "/auth/logout"
        case .getProfile: endpointPath = "/profile"
        case .deleteAccount: endpointPath = "/profile"
        case .getProfileStats: endpointPath = "/profile/stats"
        case .getPublicUserProfile(let id): endpointPath = "/users/\(id)"
        case .getVisitedLocations: endpointPath = "/profile/visited-locations"
        case .deleteAvatar: endpointPath = "/profile/avatar"
        case .updateProfile: endpointPath = "/profile"
        case .requestAvatarUpload: endpointPath = "/profile/avatar/upload"
        case .confirmAvatarUpload: endpointPath = "/profile/avatar/confirm"
        case .changeEmail: endpointPath = "/profile/change-email"
        case .confirmEmailChange: endpointPath = "/profile/confirm-email"
        case .registerDeviceToken, .unregisterDeviceToken:
            endpointPath = "/profile/device-tokens"
        case .getFeed: endpointPath = "/feed"
        case .getRecommendations: endpointPath = "/recommendations"
        case .saveRecommendation: endpointPath = "/recommendations/save"
        case .getTrips: endpointPath = "/trips"
        case .getFavouriteTrips: endpointPath = "/trips/favourites"
        case .getTrip(let id): endpointPath = "/trips/\(id)"
        case .requestTripCoverUpload(let id, _, _): endpointPath = "/trips/\(id)/cover/upload"
        case .confirmTripCoverUpload(let id, _): endpointPath = "/trips/\(id)/cover/confirm"
        case .updateTrip(let id, _, _, _, _, _, _, _, _): endpointPath = "/trips/\(id)"
        case .deleteTrip(let id): endpointPath = "/trips/\(id)"
        case .joinTripByToken: endpointPath = "/trips/join"
        case .generateInviteLink(let tripId, _): endpointPath = "/trips/\(tripId)/invite"
        case .leaveTrip(let id): endpointPath = "/trips/\(id)/leave"
        case .removeParticipant(let tripId, let userId): endpointPath = "/trips/\(tripId)/participants/\(userId)"
        case .publishTrip(let id, _, _): endpointPath = "/trips/\(id)/publish"
        case .updateTripSettings(let id, _): endpointPath = "/trips/\(id)/settings"
        case .likeTrip(let id): endpointPath = "/trips/\(id)/like"
        case .dislikeTrip(let id): endpointPath = "/trips/\(id)/dislike"
        case .addTripToFavourites(let id): endpointPath = "/trips/\(id)/favourite"
        case .removeTripFromFavourites(let id): endpointPath = "/trips/\(id)/favourite"
        case .startBattle(let tripId): endpointPath = "/trips/\(tripId)/battles"
        case let .submitBattleResult(tripId, battleId, _): endpointPath = "/trips/\(tripId)/battles/\(battleId)/result"
        case .setTripPrivacy(let tripId, _): endpointPath = "/trips/\(tripId)/privacy"
        case let .setPinPrivacy(tripId, pinId, _): endpointPath = "/trips/\(tripId)/pins/\(pinId)/privacy"
        case let .setMediaPrivacy(tripId, mediaId, _): endpointPath = "/trips/\(tripId)/media/\(mediaId)/privacy"
        case .getDesiredPlaces: endpointPath = "/profile/desired-places"
        case .createDesiredPlace: endpointPath = "/profile/desired-places"
        case .requestDesiredPlaceImageUpload: endpointPath = "/profile/desired-places/upload-url"
        case let .updateDesiredPlace(placeId, _, _, _): endpointPath = "/profile/desired-places/\(placeId)"
        case let .deleteDesiredPlace(placeId): endpointPath = "/profile/desired-places/\(placeId)"
        case let .deleteDesiredPlaceImage(placeId): endpointPath = "/profile/desired-places/\(placeId)/image"
        case let .getPin(tripId, pinId): endpointPath = "/trips/\(tripId)/pins/\(pinId)"
        case let .deletePin(tripId, pinId): endpointPath = "/trips/\(tripId)/pins/\(pinId)"
        case let .deletePinMedia(tripId, pinId, mediaId):
            endpointPath = "/trips/\(tripId)/pins/\(pinId)/media/\(mediaId)"
        case let .updatePin(tripId, pinId, _, _, _, _, _, _, _, _, _): endpointPath = "/trips/\(tripId)/pins/\(pinId)"
        case .searchPins: endpointPath = "/pins/search"
        case .createTrip: endpointPath = "/trips/creation/start"
        case .processMediaGrouping(let tripId, _): endpointPath = "/trips/creation/\(tripId)/media/process-grouping"
        case .applyGroupsAndProcess(let tripId, _, _): endpointPath = "/trips/creation/\(tripId)/apply-groups-and-process"
        case .getTripReview(let tripId): endpointPath = "/trips/creation/\(tripId)/review"
        case .finalizeTrip(let tripId, _, _): endpointPath = "/trips/creation/\(tripId)/finalize"
        case .pinUploadStart(let tripId, _, _): endpointPath = "/trips/\(tripId)/pin-uploads/start"
        case let .pinUploadRequestUploadUrls(tripId, sessionId, _): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/upload-urls"
        case let .pinUploadCommitUpload(tripId, sessionId, _, _, _, _, _): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/commit-upload"
        case let .pinUploadProcess(tripId, sessionId): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/process"
        case let .pinUploadGetReview(tripId, sessionId): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/review"
        case let .pinUploadFinalize(tripId, sessionId, _): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/finalize"
        case let .pinUploadCancel(tripId, sessionId): endpointPath = "/trips/\(tripId)/pin-uploads/\(sessionId)/cancel"
        case .addMediaStart(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/start"
        case .addMediaRequestUploadUrls(let tripId, _, _): endpointPath = "/trips/\(tripId)/media/add/request-upload-urls"
        case .addMediaCommitUpload(let tripId, _, _, _, _, _, _): endpointPath = "/trips/\(tripId)/media/add/commit-upload"
        case .addMediaGetSessionMedia(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/session-media"
        case .addMediaProcessGrouping(let tripId, _, _): endpointPath = "/trips/\(tripId)/media/add/process-grouping"
        case .addMediaGetGrouping(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/grouping"
        case .addMediaApplyGroupsAndProcess(let tripId, _, _, _): endpointPath = "/trips/\(tripId)/media/add/apply-groups-and-process"
        case .addMediaGetReview(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/review"
        case .addMediaConfirm(let tripId, _, _, _): endpointPath = "/trips/\(tripId)/media/add/confirm"
        case .addMediaCancel(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/cancel"
        case .addMediaTakeover(let tripId, _): endpointPath = "/trips/\(tripId)/media/add/takeover"
        }
        return "/api/v1\(endpointPath)"
    }

    var method: Moya.Method {
        switch self {
        case .getFeed, .getRecommendations, .getTrips, .getFavouriteTrips, .getTrip, .getTripReview:
            return .get
        case .getProfile, .getProfileStats, .getVisitedLocations, .getPublicUserProfile:
            return .get
        case .getPin, .searchPins:
            return .get
        case .addMediaGetSessionMedia, .addMediaGetGrouping, .addMediaGetReview:
            return .get
        case .pinUploadGetReview:
            return .get
        case .deleteAvatar:
            return .delete
        case .updateTrip, .updateTripSettings, .updateProfile, .updateDesiredPlace, .updatePin:
            return .patch
        case .setTripPrivacy, .setPinPrivacy, .setMediaPrivacy:
            return .put
        case .unregisterDeviceToken:
            return .delete
        case .deleteTrip, .removeParticipant, .deleteAccount, .removeTripFromFavourites,
             .deleteDesiredPlace, .deleteDesiredPlaceImage, .deletePin, .deletePinMedia:
            return .delete
        case .getDesiredPlaces:
            return .get
        default:
            return .post
        }
    }

    // swiftlint:disable:next function_body_length
    var task: Moya.Task {
        switch self {
        case .getTrips, .getTrip, .getTripReview, .deleteTrip, .removeParticipant,
             .leaveTrip, .likeTrip, .dislikeTrip, .addTripToFavourites, .removeTripFromFavourites:
            return .requestPlain
        case .getProfile, .getProfileStats, .getPublicUserProfile:
            return .requestPlain
        case let .getVisitedLocations(type):
            var params: [String: Any] = [:]
            if let type { params["type"] = type }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)

        case let .getFeed(limit, offset, category, season, city, country, sortBy):
            var params: [String: Any] = [:]
            if let limit { params["limit"] = limit }
            if let offset { params["offset"] = offset }
            if let category { params["category"] = category }
            if let season { params["season"] = season }
            if let city { params["city"] = city }
            if let country { params["country"] = country }
            if let sortBy { params["sort_by"] = sortBy }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)
        case let .getRecommendations(city, country, category, season):
            var params: [String: Any] = [:]
            if let city { params["city"] = city }
            if let country { params["country"] = country }
            if let category { params["category"] = category }
            if let season { params["season"] = season }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)
        case let .saveRecommendation(snapshotToken, pinIds, city, country, category, season):
            let params: [String: Any] = [
                "snapshot_token": snapshotToken,
                "pin_ids": pinIds,
                "city": city ?? "",
                "country": country ?? "",
                "category": category ?? "",
                "season": season ?? ""
            ]
            return .requestParameters(parameters: params, encoding: JSONEncoding.default)

        case let .getFavouriteTrips(limit, offset):
            var params: [String: Any] = [:]
            if let limit { params["limit"] = limit }
            if let offset { params["offset"] = offset }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)

        case let .submitEmail(email): return jsonParams(["email": email])
        case let .devLogin(email): return jsonParams(["email": email])
        case let .verifyEmail(regId, code): return jsonParams(["registration_id": regId, "verification_code": code])
        case let .passkeyLoginBegin(email): return jsonParams(["email": email])
        case let .passkeyLoginFinish(email, cred): return jsonParams(["email": email, "credential_json": cred])
        case let .passkeyRegisterBegin(regId, username): return jsonParams(["registration_id": regId, "username": username])
        case let .passkeyRegisterFinish(regId, cred): return jsonParams(["registration_id": regId, "credential_json": cred])
        case let .refreshToken(token): return jsonParams(["refresh_token": token])
        case let .logout(token): return jsonParams(["refresh_token": token])
        case .deleteAccount, .deleteAvatar:
            return .requestPlain

        case let .updateProfile(username): return jsonParams(["username": username])
        case let .requestAvatarUpload(filename, contentType): return jsonParams(["filename": filename, "content_type": contentType])
        case let .confirmAvatarUpload(s3Key): return jsonParams(["s3_key": s3Key])
        case let .requestTripCoverUpload(_, filename, contentType): return jsonParams(["filename": filename, "content_type": contentType])
        case let .confirmTripCoverUpload(_, s3Key): return jsonParams(["s3_key": s3Key])
        case let .changeEmail(userId, newEmail):
            var params: [String: Any] = ["new_email": newEmail]
            if let userId { params["user_id"] = userId }
            return jsonParams(params)
        case let .confirmEmailChange(code): return jsonParams(["verification_code": code])
        case let .registerDeviceToken(token): return jsonParams(["apns_token": token])
        case let .unregisterDeviceToken(token): return jsonParams(["apns_token": token])

        case let .joinTripByToken(token): return jsonParams(["token": token])
        case let .generateInviteLink(_, secs):
            var params: [String: Any] = [:]
            if let secs { params["expires_in_seconds"] = secs }
            return jsonParams(params)
        case let .publishTrip(_, whole, pinIds): return jsonParams(["publish_whole": whole, "pin_ids": pinIds])
        case .startBattle:
            return .requestPlain
        case let .submitBattleResult(_, _, winnerMediaId):
            return jsonParams(["winner_media_id": winnerMediaId])
        case let .updateTripSettings(_, enabled): return jsonParams(["notifications_enabled": enabled])

        case let .updateTrip(_, name, desc, cat, season, privacy, cover, start, end):
            var params: [String: Any] = [:]
            if let name { params["name"] = name }
            if let desc { params["description"] = desc }
            if let cat { params["category"] = cat }
            if let season { params["season"] = season }
            if let privacy { params["privacy_level"] = privacy }
            if let cover { params["cover_url"] = cover }
            if let start { params["start_date_unix"] = start }
            if let end { params["end_date_unix"] = end }
            return jsonParams(params)

        case let .createTrip(name, desc, cat, season, files):
            struct Body: Encodable {
                let name: String; let description: String?; let category: String?
                let season: String?; let files_to_upload: [FileToUploadJSON]
            }
            return .requestJSONEncodable(Body(name: name, description: desc, category: cat, season: season, files_to_upload: files.map(FileToUploadJSON.init)))

        case let .processMediaGrouping(_, media):
            struct Body: Encodable { let media: [MediaMetaEntryJSON] }
            return .requestJSONEncodable(Body(media: media.map(MediaMetaEntryJSON.init)))

        case let .applyGroupsAndProcess(_, pins, deleted):
            struct Body: Encodable { let draft_pins: [DraftPinInputJSON]; let deleted_media_ids: [String] }
            return .requestJSONEncodable(Body(draft_pins: pins.map(DraftPinInputJSON.init), deleted_media_ids: deleted))

        case let .finalizeTrip(_, updates, toDelete):
            struct Body: Encodable { let pin_updates: [PinUpdateInputJSON]; let media_to_delete: [String] }
            return .requestJSONEncodable(Body(pin_updates: updates.map(PinUpdateInputJSON.init), media_to_delete: toDelete))

        case let .pinUploadStart(_, targetPinId, files):
            return .requestJSONEncodable(
                PinUploadStartJSON(
                    target_pin_id: targetPinId,
                    files_to_upload: files.map(FileToUploadJSON.init)
                )
            )

        case let .pinUploadRequestUploadUrls(_, _, files):
            struct PinUploadRequestUploadUrlsBody: Encodable { let files_to_upload: [FileToUploadJSON] }
            return .requestJSONEncodable(PinUploadRequestUploadUrlsBody(files_to_upload: files.map(FileToUploadJSON.init)))

        case let .pinUploadCommitUpload(_, _, s3Key, mediaType, capturedAtUnix, lat, lon):
            var params: [String: Any] = ["s3_key": s3Key, "media_type": mediaType]
            if let capturedAtUnix { params["captured_at_unix"] = capturedAtUnix }
            if let lat { params["latitude"] = lat }
            if let lon { params["longitude"] = lon }
            return jsonParams(params)

        case .pinUploadProcess, .pinUploadCancel, .pinUploadGetReview:
            return .requestPlain

        case let .pinUploadFinalize(_, _, input):
            return .requestJSONEncodable(PinUploadFinalizeJSON(input))

        case let .addMediaStart(_, files):
            struct AddMediaStartBody: Encodable { let files_to_upload: [FileToUploadJSON] }
            return .requestJSONEncodable(AddMediaStartBody(files_to_upload: files.map(FileToUploadJSON.init)))

        case let .addMediaRequestUploadUrls(_, sessionId, files):
            struct AddMediaRequestUploadUrlsBody: Encodable { let session_id: String; let files_to_upload: [FileToUploadJSON] }
            return .requestJSONEncodable(AddMediaRequestUploadUrlsBody(session_id: sessionId, files_to_upload: files.map(FileToUploadJSON.init)))

        case let .addMediaCommitUpload(_, sessionId, s3Key, mediaType, capturedAt, lat, lon):
            var params: [String: Any] = ["session_id": sessionId, "s3_key": s3Key, "media_type": mediaType]
            if let capturedAt { params["captured_at"] = capturedAt }
            if let lat { params["latitude"] = lat }
            if let lon { params["longitude"] = lon }
            return jsonParams(params)

        case let .addMediaGetSessionMedia(_, sessionId):
            return .requestParameters(parameters: ["session_id": sessionId], encoding: URLEncoding.queryString)

        case let .addMediaProcessGrouping(_, sessionId, addMore):
            return jsonParams(["session_id": sessionId, "add_more": addMore])

        case let .addMediaGetGrouping(_, sessionId):
            return .requestParameters(parameters: ["session_id": sessionId], encoding: URLEncoding.queryString)

        case let .addMediaApplyGroupsAndProcess(_, sessionId, pins, deleted):
            struct AddMediaApplyBody: Encodable { let session_id: String; let draft_pins: [DraftPinInputJSON]; let deleted_media_ids: [String] }
            return .requestJSONEncodable(AddMediaApplyBody(session_id: sessionId, draft_pins: pins.map(DraftPinInputJSON.init), deleted_media_ids: deleted))

        case let .addMediaGetReview(_, sessionId):
            return .requestParameters(parameters: ["session_id": sessionId], encoding: URLEncoding.queryString)

        case let .addMediaConfirm(_, sessionId, updates, toDelete):
            struct AddMediaConfirmBody: Encodable { let session_id: String; let pin_updates: [PinUpdateInputJSON]; let media_to_delete: [String] }
            return .requestJSONEncodable(AddMediaConfirmBody(session_id: sessionId, pin_updates: updates.map(PinUpdateInputJSON.init), media_to_delete: toDelete))

        case let .addMediaCancel(_, sessionId):
            return jsonParams(["session_id": sessionId])

        case let .addMediaTakeover(_, sessionId):
            return jsonParams(["session_id": sessionId])

        case .setTripPrivacy(_, let level),
             .setPinPrivacy(_, _, let level),
             .setMediaPrivacy(_, _, let level):
            return jsonParams(["privacy_level": level])

        case .getDesiredPlaces, .deleteDesiredPlace, .deleteDesiredPlaceImage:
            return .requestPlain

        case let .createDesiredPlace(name, description, s3Key):
            var params: [String: Any] = ["name": name, "description": description]
            if let s3Key { params["s3_key"] = s3Key }
            return jsonParams(params)

        case let .requestDesiredPlaceImageUpload(filename, contentType):
            return jsonParams(["filename": filename, "content_type": contentType])

        case let .updateDesiredPlace(_, name, description, imageS3Key):
            var params: [String: Any] = ["name": name, "description": description]
            if let imageS3Key { params["image_s3_key"] = imageS3Key }
            return jsonParams(params)

        case .getPin, .deletePin, .deletePinMedia:
            return .requestPlain

        case let .searchPins(q, limit, offset):
            var params: [String: Any] = ["q": q]
            if let limit  { params["limit"]  = limit  }
            if let offset { params["offset"] = offset }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)

        case let .updatePin(_, _, name, desc, cat, lat, lon, start, end, tags, tagsSet):
            var params: [String: Any] = [:]
            if let name    { params["name"]            = name    }
            if let desc    { params["description"]     = desc    }
            if let cat     { params["category"]        = cat     }
            if let lat     { params["latitude"]        = lat     }
            if let lon     { params["longitude"]       = lon     }
            if let start   { params["start_time_unix"] = start   }
            if let end     { params["end_time_unix"]   = end     }
            if let tags    { params["tags"]            = tags    }
            if let tagsSet { params["tags_set"]        = tagsSet }
            return jsonParams(params)
        }
    }

    var headers: [String: String]? {
        switch self {
        case .submitEmail, .devLogin, .verifyEmail,
             .passkeyLoginBegin, .passkeyLoginFinish,
             .passkeyRegisterBegin, .passkeyRegisterFinish,
             .refreshToken:
            return ["Content-Type": "application/json"]
        default:
            var result: [String: String] = ["Content-Type": "application/json"]
            if let token = TokenStorage.shared.accessToken {
                result["Authorization"] = "Bearer \(token)"
            }
            return result
        }
    }

    private func jsonParams(_ parameters: [String: Any]) -> Moya.Task {
        .requestParameters(parameters: parameters, encoding: JSONEncoding.default)
    }
}

// MARK: - Stubs

// swiftlint:disable function_body_length
extension PinzAPI {
    var sampleData: Data {
        let json: String
        switch self {
        case .submitEmail:
            json = #"{"is_registered": false, "registration_id": "550e8400-e29b-41d4-a716-446655440000"}"#
        case .devLogin, .passkeyLoginFinish, .passkeyRegisterFinish:
            json = #"{"access_token": "stub_access_token", "refresh_token": "stub_refresh_token"}"#
        case .verifyEmail:
            json = #"{"success": true}"#
        case .passkeyLoginBegin, .passkeyRegisterBegin:
            json = #"{"options_json": "eyJjaGFsbGVuZ2UiOiJ0ZXN0In0="}"#
        case .refreshToken:
            json = #"{"access_token": "stub_new_access_token"}"#
        case .logout:
            json = #"{"success": true}"#
        case .getProfile:
            json = #"{"user_id":"user-001","username":"flowykk","nickname":"Flow","email":"flowykk@example.com","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}"#
        case .getProfileStats:
            json = #"{"total_trips":12,"total_pins":42,"total_media":77,"total_likes":128,"total_dislikes":4,"battles_finished":3}"#
        case .getVisitedLocations:
            json = #"[{"parent_id":null,"location_id":"ru","name":"Россия","last_visited_at_unix":1700000000,"visits_count":12},{"parent_id":null,"location_id":"fr","name":"Франция","last_visited_at_unix":1698000000,"visits_count":5}]"#
        case .deleteAccount:
            json = #"{"success": true}"#
        case .deleteAvatar:
            json = #"{"user_id":"user-001","username":"flowykk","nickname":"Flow","email":"flowykk@example.com","avatar_url":null}"#
        case .updateProfile:
            json = #"{"user_id":"user-001","username":"new_username","nickname":"Flow","email":"flowykk@example.com","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}"#
        case .requestAvatarUpload:
            json = #"{"upload_url":"https://pinz.s3.example.com/upload","s3_key":"avatars/user-001/avatar.png","expires_at_unix":1700000000}"#
        case .requestTripCoverUpload:
            json = #"{"upload_url":"https://pinz.s3.example.com/upload","s3_key":"trips/trip-001/cover.png","expires_at_unix":1700000000}"#
        case .confirmAvatarUpload:
            json = #"{"user_id":"user-001","username":"flowykk","nickname":"Flow","email":"flowykk@example.com","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}"#
        case .confirmTripCoverUpload:
            json = #"{"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу","category":"vacation","season":"spring","cover_url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":false,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}"#
        case .changeEmail:
            json = #"{"success":true,"message":"Verification code sent","email":"new@example.com","expires_at_unix":1700000000}"#
        case .confirmEmailChange:
            json = #"{"success":true,"message":"Email changed","email":"new@example.com"}"#
        case .registerDeviceToken:
            json = #"{"token_id":"550e8400-e29b-41d4-a716-446655440000"}"#
        case .unregisterDeviceToken:
            json = #"{"success":true}"#
        case .getRecommendations:
            json = #"{"map":{"snapshot_token":"stub-snapshot-token-001","media":[{"media_id":"rec-media-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","media_type":"image"}],"pins":[{"id":"rec-pin-001","trip_id":"trip-rec-001","name":"Тайные улочки Пекина","description":"Лучшие места для прогулок и съемки в историческом центре города","category":"vacation","latitude":39.9042,"longitude":116.4074,"location_name":"beijing","media_count":1,"media":[{"media_id":"rec-pin-media-001","url":"https://i.pinimg.com/1200x/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg","media_type":"image"}]}],"region_name":"beijing","region_type":"city","trip":{"id":"trip-rec-001","name":"Тайная Пекинская неделя","description":"Сборка локаций и маршрутов из одного города на один уикенд","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":73,"dislikes_count":1,"participants_count":2,"media_count":4,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}}}"#
        case .saveRecommendation:
            json = #"{"trip":{"id":"trip-rec-generated-001","name":"Популярные места: beijing","description":"Сборка локаций и маршрутов из одного города на один уикенд","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"private","status":"published","is_published":false,"is_generated":true,"likes_count":0,"dislikes_count":0,"participants_count":1,"media_count":4,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}}"#
        case let .getFeed(_, offset, _, _, _, _, _):
            switch offset ?? 0 {
            case 0:
                // Page 1: 2 items
                json = #"""
                [
                  {
                    "is_liked": true,
                    "is_disliked": false,
                    "is_saved": false,
                    "trip": {
                      "id": "trip-001",
                      "name": "Парижская романтика 18+ trash banned",
                      "description": "Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу",
                      "category": "vacation",
                      "season": "spring",
                      "cover_url": null,
                      "owner_user_id": "user-001",
                      "privacy_level": "public",
                      "status": "published",
                      "is_published": false,
                      "is_generated": false,
                      "likes_count": 42,
                      "dislikes_count": 2,
                      "participants_count": 12,
                      "media_count": 36,
                      "start_date_unix": 1700000000,
                      "end_date_unix": 1700200000,
                      "created_at_unix": 1699900000,
                      "updated_at_unix": 1699900000,
                      "name_censored": true
                    },
                    "pins": [
                      {
                        "id": "pin-feed-001",
                        "latitude": 48.8584,
                        "longitude": 2.2945,
                        "media": [
                          {
                            "media_id": "feed-001-001",
                            "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg",
                            "media_type": "image"
                          },
                          {
                            "media_id": "feed-001-002",
                            "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg",
                            "media_type": "image"
                          }
                        ]
                      }
                    ],
                    "media": [
                      {
                        "media_id": "m-feed-001",
                        "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg",
                        "media_type": "image"
                      }
                    ]
                  },
                  {
                    "is_liked": false,
                    "is_disliked": true,
                    "is_saved": true,
                    "trip": {
                      "id": "trip-002",
                      "name": "Горнолыжный тур в Альпы",
                      "description": "описание которое не пройдёт модерацию",
                      "category": "active",
                      "season": "winter",
                      "cover_url": null,
                      "owner_user_id": "user-002",
                      "privacy_level": "public",
                      "status": "published",
                      "is_published": true,
                      "is_generated": false,
                      "likes_count": 38,
                      "dislikes_count": 1,
                      "participants_count": 8,
                      "media_count": 30,
                      "start_date_unix": 1698000000,
                      "end_date_unix": 1698400000,
                      "created_at_unix": 1697900000,
                      "updated_at_unix": 1697950000,
                      "description_censored": true
                    },
                    "pins": [
                      {
                        "id": "pin-feed-004",
                        "latitude": 46.8182,
                        "longitude": 8.2275,
                        "media": [
                          {
                            "media_id": "feed-004-001",
                            "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg",
                            "media_type": "image"
                          }
                        ]
                      }
                    ],
                    "media": [
                      {
                        "media_id": "feed-alt-001",
                        "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg",
                        "media_type": "image"
                      }
                    ]
                  }
                ]
                """#
            case 2:
                // Page 2: 1 item (< pageSize=2 → triggers hasReachedEnd)
                json = #"""
                [
                  {
                    "is_liked": false,
                    "is_disliked": false,
                    "is_saved": false,
                    "trip": {
                      "id": "trip-003",
                      "name": "Сафари в Кении",
                      "description": "Удивительный мир дикой природы Африки",
                      "category": "active",
                      "season": "summer",
                      "cover_url": null,
                      "owner_user_id": "user-003",
                      "privacy_level": "public",
                      "status": "published",
                      "is_published": true,
                      "is_generated": false,
                      "likes_count": 91,
                      "dislikes_count": 3,
                      "participants_count": 4,
                      "media_count": 15,
                      "start_date_unix": 1695000000,
                      "end_date_unix": 1695600000,
                      "created_at_unix": 1694900000,
                      "updated_at_unix": 1694950000
                    },
                    "pins": [
                      {
                        "id": "pin-feed-007",
                        "latitude": -1.2921,
                        "longitude": 36.8219,
                        "media": [
                          {
                            "media_id": "feed-007-001",
                            "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg",
                            "media_type": "image"
                          }
                        ]
                      }
                    ],
                    "media": [
                      {
                        "media_id": "m-feed-007",
                        "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg",
                        "media_type": "image"
                      }
                    ]
                  }
                ]
                """#
            default:
                // No more items
                json = "[]"
            }
        case .getTrips, .getFavouriteTrips:
            json = #"""
            [
              {
                "id": "trip-001",
                "name": "Парижская романтика 18+ trash banned",
                "description": "Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу",
                "category": "vacation",
                "season": "spring",
                "cover_url": null,
                "owner_user_id": "user-001",
                "privacy_level": "public",
                "status": "published",
                "is_published": false,
                "is_generated": false,
                "likes_count": 42,
                "dislikes_count": 2,
                "participants_count": 12,
                "media_count": 36,
                "start_date_unix": 1700000000,
                "end_date_unix": 1700200000,
                "created_at_unix": 1699900000,
                "updated_at_unix": 1699900000,
                "name_censored": true,
                "description_censored": false
              },
              {
                "id": "trip-002",
                "name": "Горнолыжный тур в Альпы",
                "description": "Захватывающие спуски и потрясающие горные пейзажи в сердце Европы",
                "category": "active",
                "season": "winter",
                "cover_url": null,
                "owner_user_id": "user-002",
                "privacy_level": "public",
                "status": "published",
                "is_published": true,
                "is_generated": false,
                "likes_count": 38,
                "dislikes_count": 1,
                "participants_count": 8,
                "media_count": 30,
                "start_date_unix": 1698000000,
                "end_date_unix": 1698400000,
                "created_at_unix": 1697900000,
                "updated_at_unix": 1697950000
              },
              {
                "id": "trip-003",
                "name": "Пляжный отпуск на Бали",
                "description": "описание содержит ругательство и не пройдёт модерацию",
                "category": "vacation",
                "season": "summer",
                "cover_url": null,
                "owner_user_id": "user-003",
                "privacy_level": "public",
                "status": "published",
                "is_published": true,
                "is_generated": false,
                "likes_count": 156,
                "dislikes_count": 5,
                "participants_count": 24,
                "media_count": 52,
                "start_date_unix": 1720000000,
                "end_date_unix": 1720400000,
                "created_at_unix": 1719900000,
                "updated_at_unix": 1719950000,
                "name_censored": false,
                "description_censored": true
              }
            ]
            """#
        case .getPublicUserProfile:
            json = #"""
            {
              "id": "user-002",
              "username": "maria_k",
              "avatar_url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg",
              "created_at": 1699900000,
              "desired_places": [
                {"id": "dp-001", "name": "Токио", "description": "Мечтаю посетить японскую столицу, попробовать настоящую японскую кухню и увидеть Фудзияму.", "image_url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg", "created_at": 1699900001},
                {"id": "dp-002", "name": "Исландия", "description": "Хочу увидеть северное сияние и поплавать в Голубой лагуне.", "image_url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg", "created_at": 1699900002},
                {"id": "dp-003", "name": "Патагония", "description": "Дикая природа Южной Америки, ледники и горы Анд.", "image_url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg", "created_at": 1699900003}
              ]
            }
            """#
        case .getTrip:
            json = #"""
            {"trip":{"id":"trip-001","name":"Парижская романтика 18+","description":"очень плохое описание, такое не пройдёт модерацию","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":false,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1708992000,"end_date_unix":1709251200,"created_at_unix":1699900000,"updated_at_unix":1699900000,"name_censored":true,"description_censored":true},"participants":[{"user_id":"user-001","username":"flowykk","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg","role":"admin","privacy_level":"public"},{"user_id":"user-002","username":"maria_k","avatar_url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg","role":"member","privacy_level":"public"},{"user_id":"user-003","username":"den_explore","avatar_url":"https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg","role":"member","privacy_level":"public"}],"current_user_settings":{"notifications_enabled":true},"active_add_media_session":null,"pins":[
              {"id":"pin-001","name":"Эйфелева башня","description":"плохое описание, прошедшее цензуру","category":"entertainment","latitude":48.8584,"longitude":2.2945,"location_name":"Париж","tags":["архитектура","достопримечательность"],"issues":[],"name_censored":true,"description_censored":true,"media":[
                {"media_id":"m-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","privacy_level":"public"},
                {"media_id":"m-002","url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg","privacy_level":"public"},
                {"media_id":"m-003","url":"https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg","privacy_level":"public"},
                {"media_id":"m-004","url":"https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg","privacy_level":"public"},
                {"media_id":"m-005","url":"https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-002","name":"Лувр","description":"плохое описание которое будет заменено","category":"entertainment","latitude":48.8606,"longitude":2.3352,"location_name":"Париж","tags":["музей","искусство"],"issues":[],"description_censored":true,"media":[
                {"media_id":"m-006","url":"https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg","privacy_level":"public"},
                {"media_id":"m-007","url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg","privacy_level":"public"},
                {"media_id":"m-008","url":"https://i.pinimg.com/736x/59/79/59/5979594c0f0de1b583f60ce9ac15b94e.jpg","privacy_level":"public"},
                {"media_id":"m-009","url":"https://i.pinimg.com/736x/29/9e/ff/299effcb075e97c1b4dc5ebcb7aac061.jpg","privacy_level":"public"},
                {"media_id":"m-010","url":"https://i.pinimg.com/736x/aa/a9/1f/aaa91f5d5b7a4d2f9c2a4d57f8f0e8e0.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-003","name":"Собор Парижской Богоматери","category":"entertainment","latitude":48.8530,"longitude":2.3499,"location_name":"Париж","tags":["готика","история"],"issues":[],"media":[
                {"media_id":"m-011","url":"https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg","privacy_level":"public"},
                {"media_id":"m-012","url":"https://i.pinimg.com/1200x/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg","privacy_level":"public"},
                {"media_id":"m-013","url":"https://i.pinimg.com/736x/1f/2d/c7/1f2dc7ba98b1c5c737e8942aab90751d.jpg","privacy_level":"public"},
                {"media_id":"m-014","url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-004","name":"Монмартр","category":"entertainment","latitude":48.8867,"longitude":2.3431,"location_name":"Париж","tags":["сцена","культура"],"issues":[],"media":[
                {"media_id":"m-015","url":"https://i.pinimg.com/736x/77/65/ac/7765ac5175540792659b036142c9a49d.jpg","privacy_level":"public"},
                {"media_id":"m-016","url":"https://i.pinimg.com/736x/06/dc/fa/06dcfa6e1a3aaf1539724b3d48f21280.jpg","privacy_level":"public"},
                {"media_id":"m-017","url":"https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg","privacy_level":"public"},
                {"media_id":"m-018","url":"https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-005","name":"Сен-Шапель","category":"entertainment","latitude":48.8554,"longitude":2.3594,"location_name":"Париж","tags":["стекло","готика"],"issues":[],"media":[
                {"media_id":"m-019","url":"https://i.pinimg.com/736x/b1/d4/07/b1d4074af9450d9ce0b6f2fe5db8f36c.jpg","privacy_level":"public"},
                {"media_id":"m-020","url":"https://i.pinimg.com/1200x/83/3d/4e/833d4ec2c8b7afe0593de70d09823443.jpg","privacy_level":"public"},
                {"media_id":"m-021","url":"https://i.pinimg.com/736x/f5/ce/ef/f5ceef7cf315cb31474d66a41e093b13.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-006","name":"Елисейские поля","category":"entertainment","latitude":48.8698,"longitude":2.3078,"location_name":"Париж","tags":["шопинг","прогулка"],"issues":[],"media":[
                {"media_id":"m-022","url":"https://i.pinimg.com/736x/1c/f0/2f/1cf02f94d8800d6a172c3f4e554eb512.jpg","privacy_level":"public"},
                {"media_id":"m-023","url":"https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg","privacy_level":"public"},
                {"media_id":"m-024","url":"https://i.pinimg.com/1200x/e2/b4/26/e2b426206dbb0b1cc832c80e2d9259ee.jpg","privacy_level":"public"},
                {"media_id":"m-025","url":"https://i.pinimg.com/736x/2f/0b/16/2f0b16ad2c349d732a53b97ae30932f2.jpg","privacy_level":"public"}
              ]},
              {"id":"pin-007","name":"Остров Сите","category":"entertainment","latitude":48.8529,"longitude":2.3500,"location_name":"Париж","tags":["архитектура","история"],"issues":[],"media":[
                {"media_id":"m-026","url":"https://i.pinimg.com/1200x/5f/2e/15/5f2e1561dc3ddd63cb50435e360a6abb.jpg","privacy_level":"public"},
                {"media_id":"m-027","url":"https://i.pinimg.com/1200x/a9/e8/67/a9e867ac241af016ee06bea2cd5b5abb.jpg","privacy_level":"public"},
                {"media_id":"m-028","url":"https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg","privacy_level":"public"}
              ]}
            ]}
            """#
        case .updateTrip, .publishTrip:
            json = #"""
            {"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу. Для любителей истории и культуры - это неповторимое путешествие, полное волшебства и изящества.","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1708992000,"end_date_unix":1709251200,"created_at_unix":1699900000,"updated_at_unix":1699900000}
            """#
        case .deleteTrip, .removeParticipant:
            json = ""
        case .joinTripByToken:
            json = #"{"trip_id":"trip-001","already_joined":false}"#
        case .generateInviteLink:
            json = #"{"invite_link_id":"link-001","invite_url":"https://pinz.website/join/stub_token","token":"stub_token","expires_at_unix":1700300000}"#
        case .leaveTrip:
            json = #"{"success":true,"trip_deleted":false}"#
        case .updateTripSettings, .likeTrip, .dislikeTrip, .addTripToFavourites:
            json = #"{"success":true}"#
        case .removeTripFromFavourites:
            json = "{}"
        case .startBattle:
            json = #"""
            {
              "battle_id": "battle-001",
              "media": [
                {"media_id": "m-001", "media_type": "image", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"},
                {"media_id": "m-002", "media_type": "image", "url": "https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg"},
                {"media_id": "m-003", "media_type": "video", "url": "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"},
                {"media_id": "m-004", "media_type": "image", "url": "https://i.pinimg.com/736x/34/cb/9314/34cb93114fb0cca8f020cb9c26928394.jpg"},
                {"media_id": "m-005", "media_type": "image", "url": "https://i.pinimg.com/1200x/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg"},
                {"media_id": "m-006", "media_type": "image", "url": "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg"},
                {"media_id": "m-007", "media_type": "image", "url": "https://i.pinimg.com/736x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"},
                {"media_id": "m-008", "media_type": "image", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"}
              ]
            }
            """#
        case .submitBattleResult:
            json = #"{"success":true}"#
        case .createTrip:
            json = #"""
            {
              "trip_id": "trip-001",
              "status": "created",
              "upload_urls": [
                {"client_id": "video1", "s3_key": "trips/trip-001/video1.mp4", "url": "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"},
                {"client_id": "photo1", "s3_key": "trips/trip-001/photo1.jpg", "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"},
                {"client_id": "photo2", "s3_key": "trips/trip-001/photo2.jpg", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"},
                {"client_id": "photo3", "s3_key": "trips/trip-001/photo3.jpg", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg"},
                {"client_id": "photo4", "s3_key": "trips/trip-001/photo4.jpg", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg"},
                {"client_id": "photo5", "s3_key": "trips/trip-001/photo5.jpg", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}
              ]
            }
            """#
        case .processMediaGrouping:
            json = #"""
            {
              "trip_id": "trip-001",
              "status": "processed",
              "draft_pins": [
                {
                  "draft_pin_id": "draft-001",
                  "media": [
                    {"media_id": "media-001", "type": "image", "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"},
                    {"media_id": "media-002", "type": "image", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"}
                  ]
                },
                {
                  "draft_pin_id": "draft-002",
                  "media": [
                    {"media_id": "media-003", "type": "video", "url": "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"}
                  ]
                },
                {
                  "draft_pin_id": "draft-003",
                  "media": [
                    {"media_id": "media-004", "type": "image", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg"},
                    {"media_id": "media-005", "type": "image", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg"}
                  ]
                },
                {
                  "draft_pin_id": "draft-004",
                  "media": [
                    {"media_id": "media-006", "type": "image", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}
                  ]
                }
              ]
            }
            """#
        case .applyGroupsAndProcess:
            json = #"{"status":"processing","message":"Groups applied, processing started"}"#
        case .getTripReview:
            json = #"""
            {
              "trip_id": "trip-001",
              "status": "review",
              "similar": [],
              "pins": [
                {
                  "id": "pin-001",
                  "name": "Эйфелева башня",
                  "category": "entertainment",
                  "latitude": 48.8584,
                  "longitude": 2.2945,
                  "location_name": "Париж",
                  "tags": ["история", "архитектура", "религия"],
                  "issues": [],
                  "name_censored": true,
                  "media": [
                    {"media_id": "m-001", "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg", "privacy_level": "public"},
                    {"media_id": "m-002", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg", "privacy_level": "public"},
                    {"media_id": "m-003", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg", "privacy_level": "public"},
                    {"media_id": "m-004", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg", "privacy_level": "public"},
                    {"media_id": "m-005", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg", "privacy_level": "public"},
                    {"media_id": "m-008", "url": "https://i.pinimg.com/736x/59/79/59/5979594c0f0de1b583f60ce9ac15b94e.jpg", "privacy_level": "public"},
                    {"media_id": "m-009", "url": "https://i.pinimg.com/736x/29/9e/ff/299effcb075e97c1b4dc5ebcb7aac061.jpg", "privacy_level": "public"},
                    {"media_id": "m-010", "url": "https://i.pinimg.com/736x/aa/a9/1f/aaa91f5d5b7a4d2f9c2a4d57f8f0e8e0.jpg", "privacy_level": "public"}
                  ]
                },
                {
                  "id": "pin-002",
                  "name": "Лувр",
                  "category": "entertainment",
                  "latitude": 48.8606,
                  "longitude": 2.3352,
                  "location_name": "Париж",
                  "tags": ["достопримечательность"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-006", "url": "https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg", "privacy_level": "public"},
                    {"media_id": "m-007", "url": "https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg", "privacy_level": "public"},
                    {"media_id": "m-008", "url": "https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg", "privacy_level": "public"},
                    {"media_id": "m-009", "url": "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg", "privacy_level": "public"},
                    {"media_id": "m-011", "url": "https://i.pinimg.com/736x/70/13/e5/7013e510c6ca3a000d15989fcf12e5f0.jpg", "privacy_level": "public"}
                  ]
                },
                {
                  "id": "pin-003",
                  "name": "Собор Парижской Богоматери",
                  "category": "nature",
                  "latitude": 48.8530,
                  "longitude": 2.3499,
                  "location_name": "Париж",
                  "tags": ["достопримечательность", "религия"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-012", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg", "privacy_level": "public"},
                    {"media_id": "m-013", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg", "privacy_level": "public"},
                    {"media_id": "m-014", "url": "https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg", "privacy_level": "public"},
                    {"media_id": "m-015", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg", "privacy_level": "public"}
                  ]
                },
                {
                  "id": "pin-004",
                  "name": "Монмартр",
                  "category": "entertainment",
                  "latitude": 48.8867,
                  "longitude": 2.3431,
                  "location_name": "Париж",
                  "tags": ["культура", "театр"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-016", "url": "https://i.pinimg.com/736x/77/65/ac/7765ac5175540792659b036142c9a49d.jpg", "privacy_level": "public"},
                    {"media_id": "m-017", "url": "https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg", "privacy_level": "public"},
                    {"media_id": "m-018", "url": "https://i.pinimg.com/1200x/e2/b4/26/e2b426206dbb0b1cc832c80e2d9259ee.jpg", "privacy_level": "public"}
                  ]
                }
              ]
            }
            """#
        case .finalizeTrip:
            json = #"{"trip_id":"trip-001","status":"finalized","message":"Trip finalized successfully"}"#
        case .setTripPrivacy, .setPinPrivacy, .setMediaPrivacy:
            json = #"{"privacy_level":"public"}"#
        case .getDesiredPlaces:
            json = #"{"places":[{"id":"dp-001","name":"Токио","description":"Мечтаю посетить японскую столицу","image_url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","created_at":1699900001},{"id":"dp-002","name":"Исландия","description":"Хочу увидеть северное сияние","image_url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg","created_at":1699900002}]}"#
        case .createDesiredPlace, .updateDesiredPlace:
            json = #"{"id":"dp-new","name":"Новое место","description":"Описание","image_url":null,"created_at":1700000000}"#
        case .requestDesiredPlaceImageUpload:
            json = #"{"upload_url":"https://pinz.s3.example.com/upload","s3_key":"desired-places/user-001/stub.jpg"}"#
        case .deleteDesiredPlace, .deleteDesiredPlaceImage:
            json = #"{"success":true}"#
        case .getPin:
            json = #"{"pin":{"id":"pin-001","trip_id":"trip-001","name":"Эйфелева башня","category":"entertainment","latitude":48.8584,"longitude":2.2945,"tags":["архитектура"],"privacy_level":"public","media":[{"media_id":"m-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","media_type":"image","privacy_level":"public"}]}}"#
        case .deletePin:
            json = #"{"deletion_mode":"full"}"#
        case .deletePinMedia:
            json = #"{"pin":{"id":"pin-001","trip_id":"trip-001","name":"Эйфелева башня","category":"entertainment","latitude":48.8584,"longitude":2.2945,"tags":["архитектура"],"privacy_level":"public","media":[]}}"#
        case .updatePin:
            json = #"{"pin":{"id":"pin-001","trip_id":"trip-001","name":"Обновлённый пин","category":"entertainment","latitude":48.8584,"longitude":2.2945,"tags":["архитектура"],"privacy_level":"public","media":[{"media_id":"m-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","media_type":"image","privacy_level":"public"}]}}"#
        case .searchPins:
            json = #"[{"id":"pin-001","trip_id":"trip-001","name":"Эйфелева башня","category":"entertainment","latitude":48.8584,"longitude":2.2945,"tags":["архитектура"],"privacy_level":"public","media":[]}]"#
        case .pinUploadStart:
            json = #"{"session_id":"pin-session-001","upload_urls":[{"client_id":"photo1","s3_key":"trips/trip-001/pins/photo1.jpg","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"}]}"#
        case .pinUploadRequestUploadUrls:
            json = #"{"upload_urls":[{"client_id":"photo2","s3_key":"trips/trip-001/pins/photo2.jpg","url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"}]}"#
        case .pinUploadCommitUpload:
            json = #"{"media_id":"pin-media-001","media_count_in_session":1}"#
        case .pinUploadProcess:
            json = #"{"session_id":"pin-session-001","processing_status":"PROCESSING"}"#
        case .pinUploadGetReview:
            json = #"""
            {
              "session_id":"pin-session-001",
              "processing_status":"READY_FOR_REVIEW",
              "draft":{
                "suggested":{
                  "name":"Другое",
                  "category":"Другое",
                  "tags":null,
                  "latitude":59.9386,
                  "longitude":30.3141
                },
                "media":[
                  {"media_id":"pin-media-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","privacy_level":"private"}
                ],
                "pin_issues":["MISSING_DATES"],
                "nsfw_media_ids":null,
                "deduped_media_ids":null
              },
              "similar":null
            }
            """#
        case .pinUploadFinalize:
            json = #"{"pin":{"id":"pin-new-001","trip_id":"trip-001","name":"Эрмитаж","description":null,"category":"entertainment","latitude":59.94,"longitude":30.32,"start_time_unix":1778338000,"end_time_unix":1778341600,"tags":["музей"],"privacy_level":"public","media":[{"media_id":"pin-media-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","privacy_level":"public"}]}}"#
        case .pinUploadCancel:
            json = #"{"status":"cancelled"}"#
        case .addMediaStart:
            json = #"{"session_id":"session-001","status":"ADD_MEDIA_UPLOADING","joined":false,"upload_urls":[{"client_id":"photo1","s3_key":"trips/trip-001/photo1.jpg","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"}]}"#
        case .addMediaRequestUploadUrls:
            json = #"{"upload_urls":[{"client_id":"photo2","s3_key":"trips/trip-001/photo2.jpg","url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"}]}"#
        case .addMediaCommitUpload:
            json = #"{"media_id":"media-new-001","media_count_in_session":3,"remaining_slots":497}"#
        case .addMediaGetSessionMedia:
            json = #"{"session_id":"session-001","media":[{"media_id":"media-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","type":"image","actor_user_id":"user-001","uploaded_at":"2026-04-15T12:30:00Z"}],"media_count_in_session":1}"#
        case .addMediaProcessGrouping, .addMediaGetGrouping:
            json = #"{"trip_id":"trip-001","session_id":"session-001","status":"ADD_MEDIA_GROUPING_REVIEW","draft_pins":[{"draft_pin_id":"cluster-1","media":[{"media_id":"media-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","type":"image"}]}],"existing_media_ids":[]}"#
        case .addMediaApplyGroupsAndProcess:
            // Стаб не эмулирует WS: сразу финальный статус, клиент уйдёт на review по телу ответа.
            json = #"{"message":"Stub ready for review","status":"ADD_MEDIA_DRAFT_FINAL_REVIEW"}"#
        case .addMediaGetReview:
            json = #"""
            {
              "trip_id":"trip-001",
              "session_id":"session-001",
              "pins":[
                {
                  "id":"pin-001",
                  "trip_id":"trip-001",
                  "name":"Эйфелева башня",
                  "category":"entertainment",
                  "latitude":48.8584,
                  "longitude":2.2945,
                  "start_time_unix":1716048000,
                  "end_time_unix":1716134400,
                  "tags":["архитектура"],
                  "issues":[],
                  "privacy_level":"public",
                  "media":[{"media_id":"m-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","privacy_level":"public"}]
                },
                {
                  "id":"pin-002",
                  "trip_id":"trip-001",
                  "name":"Новый кластер",
                  "category":"sight",
                  "latitude":48.86,
                  "longitude":2.30,
                  "tags":[],
                  "issues":["MISSING_DATES"],
                  "privacy_level":"public",
                  "media":[{"media_id":"m-002","url":"https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg","privacy_level":"public"}]
                }
              ],
              "new_pin_ids":["pin-002"],
              "protected_media_ids":[],
              "current_initiator":{"user_id":"user-001","username":"flowykk","avatar_url":null},
              "takeover_available_at":"2026-04-25T13:00:00Z",
              "can_edit":true
            }
            """#
        case .addMediaConfirm:
            json = #"{"status":"READY","already_confirmed":false}"#
        case .addMediaCancel:
            json = #"{"status":"READY"}"#
        case .addMediaTakeover:
            json = #"{"is_initiator":true,"current_initiator":{"user_id":"user-001","username":"flowykk","avatar_url":null},"takeover_available_at":"2026-04-25T14:00:00Z"}"#
        }
        return json.data(using: .utf8) ?? Data()
    }
}
// swiftlint:enable function_body_length
// swiftlint:enable file_length

// MARK: - String helpers

extension String {
    func defaultUTF8Data() -> Data? { self.data(using: .utf8) }

    func toISO8601String() -> String? {
        let fmt = DateFormatter()
        fmt.locale = Locale(identifier: "en_US_POSIX")
        fmt.dateFormat = "dd.MM.yyyy HH:mm"
        guard let date = fmt.date(from: self) else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return iso.string(from: date)
    }
}

// MARK: - Session invalidation (401 / refresh failure)

/// When `true`, a 401 is being handled by `NetworkService.retryOnUnauthorized` and must not trigger a full session reset yet.
enum PinzUnauthorizedRetryContext {
    @TaskLocal static var isActive = false
}

enum PinzSessionInvalidation {

    /// Clears stored tokens and notifies the app to return to the authentication root (e.g. `AuthFlowView`).
    static func invalidateSession() {
        _Concurrency.Task { @MainActor in
            TokenStorage.shared.clear()
            NotificationCenter.default.post(name: .pinzSessionInvalidated, object: nil)
        }
    }

    static func handleUnauthorizedFromAPI(for target: PinzAPI) {
        guard target.shouldInvalidateSessionOnUnauthorized else { return }
        guard TokenStorage.shared.accessToken != nil || TokenStorage.shared.refreshToken != nil else { return }
        guard !PinzUnauthorizedRetryContext.isActive else { return }
        invalidateSession()
    }
}

private extension PinzAPI {
    /// Endpoints where 401 means a failed auth step, not an expired session.
    var shouldInvalidateSessionOnUnauthorized: Bool {
        switch self {
        case .devLogin, .submitEmail, .verifyEmail,
             .passkeyLoginBegin, .passkeyLoginFinish,
             .passkeyRegisterBegin, .passkeyRegisterFinish:
            return false
        default:
            return true
        }
    }
}
