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
    case deleteAccount
    case updateProfile(username: String)
    case requestAvatarUpload(filename: String, contentType: String)
    case confirmAvatarUpload(s3Key: String)
    case changeEmail(userId: String?, newEmail: String)
    case confirmEmailChange(verificationCode: String)

    // Feed
    case getFeed(limit: Int?, offset: Int?, category: String?, season: String?, locationId: Int?, sortBy: String?)

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
}

// MARK: - TargetType

extension PinzAPI: TargetType {
    var baseURL: URL {
        if CommandLine.arguments.contains("-useLocalhost") {
            return URL(string: "http://localhost:8080")!
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
        case .updateProfile: endpointPath = "/profile"
        case .requestAvatarUpload: endpointPath = "/profile/avatar/upload"
        case .confirmAvatarUpload: endpointPath = "/profile/avatar/confirm"
        case .changeEmail: endpointPath = "/profile/change-email"
        case .confirmEmailChange: endpointPath = "/profile/confirm-email"
        case .getFeed: endpointPath = "/feed"
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
        case .createTrip: endpointPath = "/trips/creation/start"
        case .processMediaGrouping(let tripId, _): endpointPath = "/trips/creation/\(tripId)/media/process-grouping"
        case .applyGroupsAndProcess(let tripId, _, _): endpointPath = "/trips/creation/\(tripId)/apply-groups-and-process"
        case .getTripReview(let tripId): endpointPath = "/trips/creation/\(tripId)/review"
        case .finalizeTrip(let tripId, _, _): endpointPath = "/trips/creation/\(tripId)/finalize"
        }
        return "/api/v1\(endpointPath)"
    }

    var method: Moya.Method {
        switch self {
        case .getFeed, .getTrips, .getFavouriteTrips, .getTrip, .getTripReview:
            return .get
        case .getProfile:
            return .get
        case .updateTrip, .updateTripSettings, .updateProfile:
            return .patch
        case .deleteTrip, .removeParticipant, .deleteAccount, .removeTripFromFavourites:
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

        case let .getFeed(limit, offset, category, season, locationId, sortBy):
            var params: [String: Any] = [:]
            if let limit { params["limit"] = limit }
            if let offset { params["offset"] = offset }
            if let category { params["category"] = category }
            if let season { params["season"] = season }
            if let locationId { params["location_id"] = locationId }
            if let sortBy { params["sort_by"] = sortBy }
            return .requestParameters(parameters: params, encoding: URLEncoding.queryString)

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
        case .getProfile, .deleteAccount:
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

        case let .joinTripByToken(token): return jsonParams(["token": token])
        case let .generateInviteLink(_, secs):
            var params: [String: Any] = [:]
            if let secs { params["expires_in_seconds"] = secs }
            return jsonParams(params)
        case let .publishTrip(_, whole, pinIds): return jsonParams(["publish_whole": whole, "pin_ids": pinIds])
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
        case .deleteAccount:
            json = #"{"success": true}"#
        case .updateProfile:
            json = #"{"user_id":"user-001","username":"new_username","nickname":"Flow","email":"flowykk@example.com","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}"#
        case .requestAvatarUpload:
            json = #"{"upload_url":"https://pinz.s3.example.com/upload","s3_key":"avatars/user-001/avatar.png","expires_at_unix":1700000000}"#
        case .requestTripCoverUpload:
            json = #"{"upload_url":"https://pinz.s3.example.com/upload","s3_key":"trips/trip-001/cover.png","expires_at_unix":1700000000}"#
        case .confirmAvatarUpload:
            json = #"{"user_id":"user-001","username":"flowykk","nickname":"Flow","email":"flowykk@example.com","avatar_url":"https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg"}"#
        case .confirmTripCoverUpload:
            json = #"{"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу","category":"vacation","season":"spring","cover_url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000}"#
        case .changeEmail:
            json = #"{"success":true,"message":"Verification code sent","email":"new@example.com","expires_at_unix":1700000000}"#
        case .confirmEmailChange:
            json = #"{"success":true,"message":"Email changed","email":"new@example.com"}"#
        case .getFeed, .getTrips, .getFavouriteTrips:
            json = #"""
            [
              {"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1700000000,"end_date_unix":1700200000,"created_at_unix":1699900000,"updated_at_unix":1699900000},
              {"id":"trip-002","name":"Горнолыжный тур в Альпы","description":"Захватывающие спуски и потрясающие горные пейзажи в сердце Европы","category":"active","season":"winter","cover_url":null,"owner_user_id":"user-002","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":38,"dislikes_count":1,"start_date_unix":1698000000,"end_date_unix":1698400000,"created_at_unix":1697900000,"updated_at_unix":1697950000},
              {"id":"trip-003","name":"Пляжный отпуск на Бали","description":"Тропические пляжи, рисовые террасы и древние храмы Индонезии","category":"vacation","season":"summer","cover_url":null,"owner_user_id":"user-003","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":156,"dislikes_count":5,"start_date_unix":1720000000,"end_date_unix":1720400000,"created_at_unix":1719900000,"updated_at_unix":1719950000},
              {"id":"trip-004","name":"Культурный тур по Италии","description":"Искусство, история и изысканная кухня в Риме, Флоренции и Венеции","category":"education","season":"autumn","cover_url":null,"owner_user_id":"user-004","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":97,"dislikes_count":3,"start_date_unix":1727000000,"end_date_unix":1727400000,"created_at_unix":1726900000,"updated_at_unix":1726950000},
              {"id":"trip-005","name":"Деловая поездка в Токио","description":"Современные небоскребы, традиционные святилища и безумный темп мегаполиса","category":"business","season":"spring","cover_url":null,"owner_user_id":"user-005","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":28,"dislikes_count":0,"start_date_unix":1710000000,"end_date_unix":1710200000,"created_at_unix":1709900000,"updated_at_unix":1709950000},
              {"id":"trip-006","name":"Северное сияние в Норвегии","description":"Магическое северное сияние и суровая природа Арктики","category":"vacation","season":"winter","cover_url":null,"owner_user_id":"user-006","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":203,"dislikes_count":4,"start_date_unix":1704000000,"end_date_unix":1704300000,"created_at_unix":1703900000,"updated_at_unix":1703950000},
              {"id":"trip-007","name":"Сафари в Кении","description":"Большая пятёрка, национальные парки и встреча с дикой природой","category":"active","season":"summer","cover_url":null,"owner_user_id":"user-007","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":124,"dislikes_count":2,"start_date_unix":1721000000,"end_date_unix":1721300000,"created_at_unix":1720900000,"updated_at_unix":1720950000},
              {"id":"trip-008","name":"Морской круиз по Средиземному морю","description":"Греческие острова, итальянское побережье и кристально чистая вода","category":"vacation","season":"summer","cover_url":null,"owner_user_id":"user-008","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":91,"dislikes_count":1,"start_date_unix":1722000000,"end_date_unix":1722400000,"created_at_unix":1721900000,"updated_at_unix":1721950000}
            ]
            """#
        case .getTrip:
            json = #"""
            {"trip":{"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу. Для любителей истории и культуры - это неповторимое путешествие, полное волшебства и изящества.","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1708992000,"end_date_unix":1709251200,"created_at_unix":1699900000,"updated_at_unix":1699900000},"pins":[{"id":"pin-001","name":"Эйфелева башня","category":"entertainment","latitude":48.8584,"longitude":2.2945,"location_name":"Париж","tags":["архитектура","достопримечательность"],"issues":[],"media":[{"media_id":"m-001","url":"https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg","privacy_level":"public"}]},{"id":"pin-002","name":"Лувр","category":"entertainment","latitude":48.8606,"longitude":2.3352,"location_name":"Париж","tags":["музей","искусство"],"issues":[],"media":[{"media_id":"m-003","url":"https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg","privacy_level":"public"}]},{"id":"pin-003","name":"Собор Парижской Богоматери","category":"entertainment","latitude":48.8530,"longitude":2.3499,"location_name":"Париж","tags":["готика","история"],"issues":[],"media":[{"media_id":"m-004","url":"https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg","privacy_level":"public"}]}]}
            """#
        case .updateTrip, .publishTrip:
            json = #"""
            {"id":"trip-001","name":"Парижская романтика","description":"Волшебные улицы Парижа, Эйфелева башня и уютные кафе на левом берегу. Для любителей истории и культуры - это неповторимое путешествие, полное волшебства и изящества.","category":"vacation","season":"spring","cover_url":null,"owner_user_id":"user-001","privacy_level":"public","status":"published","is_published":true,"is_generated":false,"likes_count":42,"dislikes_count":2,"start_date_unix":1708992000,"end_date_unix":1709251200,"created_at_unix":1699900000,"updated_at_unix":1699900000}
            """#
        case .deleteTrip, .removeParticipant, .removeTripFromFavourites:
            json = ""
        case .joinTripByToken:
            json = #"{"trip_id":"trip-001","already_joined":false}"#
        case .generateInviteLink:
            json = #"{"invite_link_id":"link-001","invite_url":"https://pinz.website/join/stub_token","token":"stub_token","expires_at_unix":1700300000}"#
        case .leaveTrip:
            json = #"{"success":true,"trip_deleted":false}"#
        case .updateTripSettings, .likeTrip, .dislikeTrip, .addTripToFavourites:
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
                  "id": "pin-002",
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
                  "id": "pin-003",
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
