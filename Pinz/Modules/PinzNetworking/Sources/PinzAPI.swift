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

    // Feed
    case getFeed(limit: Int?, offset: Int?, category: String?, season: String?, locationId: Int?, locationName: String?, sortBy: String?)

    // Trips CRUD
    case getTrips
    case getTrip(id: String)
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
    case transferAdmin(id: String, newAdminUserId: String)
    case likeTrip(id: String)
    case dislikeTrip(id: String)
    case addTripToFavourites(id: String)
    case removeTripFromFavourites(id: String)

    // Add-media flow
    case addMediaStart(tripId: String, filesToUpload: [FileToUploadDTO])
    case addMediaProcessGrouping(tripId: String, sessionId: String, media: [MediaMetaEntryDTO])
    case addMediaApplyGroupsAndProcess(tripId: String, sessionId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String])

    // Trip creation flow
    case createTrip(name: String, description: String?, category: String?, season: String?, filesToUpload: [FileToUploadDTO])
    case processMediaGrouping(tripId: String, media: [MediaMetaEntryDTO])
    case applyGroupsAndProcess(tripId: String, draftPins: [DraftPinInputDTO], deletedMediaIds: [String])
    case getTripReview(tripId: String)
    case finalizeTrip(tripId: String, pinUpdates: [PinUpdateInputDTO], mediaToDelete: [String])
}

// MARK: - TargetType

extension PinzAPI: TargetType {
    var baseURL: URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "http://localhost:8080")!
        }
        return URL(string: "https://pinz.website/api/v1")!
    }

    var path: String {
        switch self {
        case .devLogin: return "/auth/dev-login"
        case .submitEmail: return "/auth/email"
        case .verifyEmail: return "/auth/verify-email"
        case .passkeyLoginBegin: return "/auth/passkey/login/begin"
        case .passkeyLoginFinish: return "/auth/passkey/login/finish"
        case .passkeyRegisterBegin: return "/auth/passkey/register/begin"
        case .passkeyRegisterFinish: return "/auth/passkey/register/finish"
        case .refreshToken: return "/auth/refresh"
        case .logout: return "/auth/logout"
        case .getFeed: return "/feed"
        case .getTrips: return "/trips"
        case .getTrip(let id): return "/trips/\(id)"
        case .updateTrip(let id, _, _, _, _, _, _, _, _): return "/trips/\(id)"
        case .deleteTrip(let id): return "/trips/\(id)"
        case .joinTripByToken: return "/trips/join"
        case .generateInviteLink(let tripId, _): return "/trips/\(tripId)/invite"
        case .leaveTrip(let id): return "/trips/\(id)/leave"
        case .removeParticipant(let tripId, let userId): return "/trips/\(tripId)/participants/\(userId)"
        case .publishTrip(let id, _, _): return "/trips/\(id)/publish"
        case .updateTripSettings(let id, _): return "/trips/\(id)/settings"
        case .transferAdmin(let id, _): return "/trips/\(id)/transfer-admin"
        case .likeTrip(let id): return "/trips/\(id)/like"
        case .dislikeTrip(let id): return "/trips/\(id)/dislike"
        case .addTripToFavourites(let id): return "/trips/\(id)/favourite"
        case .removeTripFromFavourites(let id): return "/trips/\(id)/favourite"
        case .addMediaStart(let tripId, _): return "/trips/\(tripId)/media/add/start"
        case .addMediaProcessGrouping(let tripId, _, _): return "/trips/\(tripId)/media/add/process-grouping"
        case .addMediaApplyGroupsAndProcess(let tripId, _, _, _): return "/trips/\(tripId)/media/add/apply-groups-and-process"
        case .createTrip: return "/trips/creation/start"
        case .processMediaGrouping(let tripId, _): return "/trips/creation/\(tripId)/media/process-grouping"
        case .applyGroupsAndProcess(let tripId, _, _): return "/trips/creation/\(tripId)/apply-groups-and-process"
        case .getTripReview(let tripId): return "/trips/creation/\(tripId)/review"
        case .finalizeTrip(let tripId, _, _): return "/trips/creation/\(tripId)/finalize"
        }
    }

    var method: Moya.Method {
        switch self {
        case .getFeed, .getTrips, .getTrip, .getTripReview:
            return .get
        case .updateTrip, .updateTripSettings:
            return .patch
        case .deleteTrip, .removeParticipant, .removeTripFromFavourites:
            return .delete
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

        case let .getFeed(limit, offset, category, season, locationId, locationName, sortBy):
            var params: [String: Any] = [:]
            if let limit { params["limit"] = limit }
            if let offset { params["offset"] = offset }
            if let category { params["category"] = category }
            if let season { params["season"] = season }
            if let locationId { params["location_id"] = locationId }
            if let locationName { params["location_name"] = locationName }
            if let sortBy { params["sort_by"] = sortBy }
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

        case let .joinTripByToken(token): return jsonParams(["token": token])
        case let .generateInviteLink(_, secs):
            var params: [String: Any] = [:]
            if let secs { params["expires_in_seconds"] = secs }
            return jsonParams(params)
        case let .publishTrip(_, whole, pinIds): return jsonParams(["publish_whole": whole, "pin_ids": pinIds])
        case let .updateTripSettings(_, enabled): return jsonParams(["notifications_enabled": enabled])
        case let .transferAdmin(_, userId): return jsonParams(["new_admin_user_id": userId])

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

        case let .addMediaStart(_, files):
            struct Body: Encodable { let files_to_upload: [FileToUploadJSON] }
            return .requestJSONEncodable(Body(files_to_upload: files.map(FileToUploadJSON.init)))

        case let .addMediaProcessGrouping(_, sessionId, media):
            struct Body: Encodable { let session_id: String; let media: [MediaMetaEntryJSON] }
            return .requestJSONEncodable(Body(session_id: sessionId, media: media.map(MediaMetaEntryJSON.init)))

        case let .addMediaApplyGroupsAndProcess(_, sessionId, pins, deleted):
            struct Body: Encodable { let session_id: String; let draft_pins: [DraftPinInputJSON]; let deleted_media_ids: [String] }
            return .requestJSONEncodable(Body(session_id: sessionId, draft_pins: pins.map(DraftPinInputJSON.init), deleted_media_ids: deleted))
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
        case .getFeed, .getTrips:
            json = #"""
            [{"id":"trip-001","name":"Paris Trip","description":"A lovely trip","category":"city","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":12,"dislikes_count":0,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}]
            """#
        case .getTrip, .updateTrip, .publishTrip:
            json = #"""
            {"id":"trip-001","name":"Paris Trip","description":"A lovely trip","category":"city","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":12,"dislikes_count":0,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}
            """#
        case .deleteTrip, .removeParticipant, .removeTripFromFavourites:
            json = ""
        case .joinTripByToken:
            json = #"{"trip_id":"trip-001","already_joined":false}"#
        case .generateInviteLink:
            json = #"{"invite_link_id":"link-001","invite_url":"https://pinz.website/join/stub_token","token":"stub_token","expires_at_unix":1700300000}"#
        case .leaveTrip:
            json = #"{"success":true,"trip_deleted":false}"#
        case .updateTripSettings, .transferAdmin, .likeTrip, .dislikeTrip, .addTripToFavourites:
            json = #"{"success":true}"#
        case .createTrip, .addMediaStart:
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
        case .processMediaGrouping, .addMediaProcessGrouping:
            json = #"""
            {
              "trip_id": "trip-001",
              "status": "processed",
              "draft_pins": [
                {
                  "draft_pin_id": "draft-001",
                  "media": [
                    {"media_id": "media-001", "type": "photo", "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"},
                    {"media_id": "media-002", "type": "photo", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg"}
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
                    {"media_id": "media-004", "type": "photo", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg"},
                    {"media_id": "media-005", "type": "photo", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg"}
                  ]
                },
                {
                  "draft_pin_id": "draft-004",
                  "media": [
                    {"media_id": "media-006", "type": "photo", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}
                  ]
                }
              ]
            }
            """#
        case .applyGroupsAndProcess, .addMediaApplyGroupsAndProcess:
            json = #"{"status":"processing","message":"Groups applied, processing started"}"#
        case .getTripReview:
            json = #"""
            {
              "trip_id": "trip-001",
              "status": "review",
              "similar": [],
              "pins": [
                {
                  "pin_id": "pin-001",
                  "name": "Храм Христа Спасителя",
                  "category": "entertainment",
                  "latitude": 55.7447,
                  "longitude": 37.6055,
                  "location_name": "Москва",
                  "tags": ["история", "архитектура", "религия"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-001", "url": "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg", "privacy_level": "public"},
                    {"media_id": "m-002", "url": "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg", "privacy_level": "public"},
                    {"media_id": "m-003", "url": "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg", "privacy_level": "public"},
                    {"media_id": "m-004", "url": "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg", "privacy_level": "public"},
                    {"media_id": "m-005", "url": "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg", "privacy_level": "public"}
                  ]
                },
                {
                  "pin_id": "pin-002",
                  "name": "Красная площадь",
                  "category": "entertainment",
                  "latitude": 55.7539,
                  "longitude": 37.6208,
                  "location_name": "Москва",
                  "tags": ["достопримечательность"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-006", "url": "https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg", "privacy_level": "public"},
                    {"media_id": "m-007", "url": "https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg", "privacy_level": "public"},
                    {"media_id": "m-008", "url": "https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg", "privacy_level": "public"},
                    {"media_id": "m-009", "url": "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg", "privacy_level": "public"}
                  ]
                },
                {
                  "pin_id": "pin-003",
                  "name": "Парк Горького",
                  "category": "nature",
                  "latitude": 55.7312,
                  "longitude": 37.6014,
                  "location_name": "Москва",
                  "tags": ["парк", "природа"],
                  "issues": [],
                  "media": [
                    {"media_id": "m-010", "url": "https://i.pinimg.com/1200x/cd/47/23/cd4723e7bac0a34506e84b9c378d9eaf.jpg", "privacy_level": "public"},
                    {"media_id": "m-011", "url": "https://i.pinimg.com/1200x/a9/e8/67/a9e867ac241af016ee06bea2cd5b5abb.jpg", "privacy_level": "public"},
                    {"media_id": "m-012", "url": "https://i.pinimg.com/1200x/66/a0/94/66a094638921cfd9e7a3ce009bc43409.jpg", "privacy_level": "public"}
                  ]
                }
              ]
            }
            """#
        case .finalizeTrip:
            json = #"{"trip_id":"trip-001","status":"finalized","message":"Trip finalized successfully"}"#
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
