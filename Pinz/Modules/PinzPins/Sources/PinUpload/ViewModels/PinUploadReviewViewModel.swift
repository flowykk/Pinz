import SwiftUI
import CoreLocation
import PinzUI
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor
@Observable
final class PinUploadReviewViewModel {

    public enum State: SegmentedItem {
        public var id: Self { self }

        case info
        case gallery

        public var content: SegmentedItemContent {
            switch self {
            case .info: .text(PinzBaseStrings.Common.Label.info)
            case .gallery: .text(PinzBaseStrings.Common.Label.gallery)
            }
        }
    }

    enum Route {
        case back
    }

    enum Intent {
        case addTag(MediaTag)
        case deleteTag(MediaTag)
        case toggleDeleteMedia(String)
        case navigate(Route)
    }

    enum AsyncIntent {
        case reload
        case finalize
        case cancel
    }

    // Backend limits — duplicated client-side to fail fast before a known-bad request.
    static let maxNameUTF8Bytes: Int = 100
    static let maxDescriptionUTF8Bytes: Int = 5000
    static let maxTagUTF8Bytes: Int = 15
    static let maxTagsCount: Int = 10

    let tripId: String
    let sessionId: String

    var state: State = .info
    var name: String = ""
    var description: String?
    var category: PinCategory = .custom()
    var startDate: Date?
    var endDate: Date?
    var coordinates: CLLocationCoordinate2D?
    var medias: [ReviewPinMediaDTO] = []
    var tags: [MediaTag] = []
    var mediaToDelete: Set<String> = []
    var pinIssues: [String] = []
    private(set) var isLoading: Bool = false
    private(set) var initialLoaded: Bool = false

    private let networkService: NetworkServiceProtocol
    private var router: AppRouting?
    private var showToast: ((String) -> Void)?

    init(
        tripId: String,
        sessionId: String,
        networkService: NetworkServiceProtocol = NetworkService.shared
    ) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.networkService = networkService
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    func setShowToast(_ showToast: ((String) -> Void)?) {
        self.showToast = showToast
    }

    // MARK: - dispatch

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .addTag(tag):
            guard tags.count < Self.maxTagsCount else {
                showToast?("Максимум \(Self.maxTagsCount) тегов")
                return
            }
            guard tag.tag.utf8.count <= Self.maxTagUTF8Bytes else {
                showToast?("Тег слишком длинный (макс \(Self.maxTagUTF8Bytes) байт)")
                return
            }
            tags.append(tag)
        case let .deleteTag(tag):
            tags.removeAll { $0.tag == tag.tag }
        case let .toggleDeleteMedia(mediaId):
            if mediaToDelete.contains(mediaId) {
                mediaToDelete.remove(mediaId)
            } else {
                mediaToDelete.insert(mediaId)
            }
        }
    }

    // MARK: - asyncDispatch

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .reload:
            isLoading = true
            defer { isLoading = false }
            let response = try await networkService.pinUploadGetReview(
                tripId: tripId,
                sessionId: sessionId
            )
            guard
                response.processingStatus.uppercased() == "READY_FOR_REVIEW",
                let draft = response.draft
            else {
                return
            }
            applySuggested(draft.suggested)
            medias = draft.media ?? []
            pinIssues = draft.pinIssues ?? []
            initialLoaded = true

        case .finalize:
            guard validate() else { return }

            // Cannot finalize an empty pin — backend will return 409 otherwise.
            let remainingMedia = medias.filter { !mediaToDelete.contains($0.mediaId) }
            guard !remainingMedia.isEmpty else {
                showToast?("Удалены все медиа — нельзя сохранить пустой пин")
                return
            }

            isLoading = true
            defer { isLoading = false }

            let input = PinUploadFinalizeInputDTO(
                name: name.nonEmptyOrNil,
                description: description?.nonEmptyOrNil,
                category: category.apiValue,
                latitude: coordinates?.latitude,
                longitude: coordinates?.longitude,
                startTimeUnix: startDate.map { Int($0.timeIntervalSince1970) },
                endTimeUnix: endDate.map { Int($0.timeIntervalSince1970) },
                tags: tags.map(\.tag),
                tagsSet: true,
                mediaToDelete: Array(mediaToDelete)
            )

            do {
                _ = try await networkService.pinUploadFinalize(
                    tripId: tripId,
                    sessionId: sessionId,
                    input: input
                )
                PinUploadSessionStorage.shared.clear(forTripId: tripId)
                router?.notifyTripPinsReload(tripId: tripId)
                router?.popToRoot()
            } catch let httpError as HTTPError {
                handleFinalizeError(httpError)
                throw httpError
            } catch {
                showToast?("Не удалось сохранить пин")
                throw error
            }

        case .cancel:
            isLoading = true
            defer { isLoading = false }
            do {
                try await networkService.pinUploadCancel(tripId: tripId, sessionId: sessionId)
            } catch {
                // 409 / 404 — сессия уже мертва на бэке, локально дочищаем и выходим.
            }
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router?.popToRoot()
        }
    }

    // MARK: - Helpers

    private func applySuggested(_ suggested: PinSuggestedFieldsDTO?) {
        guard let suggested else { return }
        if name.isEmpty, let suggestedName = suggested.name {
            name = suggestedName
        }
        if case let .custom(value) = category, (value ?? "").isEmpty, let cat = suggested.category {
            category = cat.toPinCategory()
        }
        if startDate == nil, let unix = suggested.startTimeUnix {
            startDate = Date(timeIntervalSince1970: TimeInterval(unix))
        }
        if endDate == nil, let unix = suggested.endTimeUnix {
            endDate = Date(timeIntervalSince1970: TimeInterval(unix))
        }
        if coordinates == nil, let lat = suggested.latitude, let lon = suggested.longitude {
            coordinates = CLLocationCoordinate2D(latitude: lat, longitude: lon)
        }
        if tags.isEmpty, let serverTags = suggested.tags {
            tags = serverTags.map { MediaTag(tag: $0) }
        }
    }

    private func validate() -> Bool {
        if name.utf8.count > Self.maxNameUTF8Bytes {
            showToast?("Название слишком длинное (макс \(Self.maxNameUTF8Bytes) байт)")
            return false
        }
        if let desc = description, desc.utf8.count > Self.maxDescriptionUTF8Bytes {
            showToast?("Описание слишком длинное (макс \(Self.maxDescriptionUTF8Bytes) байт)")
            return false
        }
        if tags.count > Self.maxTagsCount {
            showToast?("Максимум \(Self.maxTagsCount) тегов")
            return false
        }
        if let bigTag = tags.first(where: { $0.tag.utf8.count > Self.maxTagUTF8Bytes }) {
            showToast?("Тег '\(bigTag.tag)' слишком длинный (макс \(Self.maxTagUTF8Bytes) байт)")
            return false
        }
        return true
    }

    private func handleFinalizeError(_ error: HTTPError) {
        switch error {
        case .conflict:
            // 409: backend can return either "pin must contain at least one media" or
            // "expected READY_FOR_REVIEW" (race with WS / parallel cancel).
            // На клиенте мы уже валидируем пустоту, так что это race — сессия мертва.
            showToast?("Сессия больше не активна")
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router?.popToRoot()
        case .badRequest:
            showToast?("Проверь поля: возможно превышены лимиты или неверные координаты")
        case .unprocessableEntity:
            // 422 LIMIT_EXCEEDED.
            showToast?("Превышен лимит — попробуй позже")
        case .preconditionFailed:
            // 412 — состояние сессии не подходит.
            showToast?("Сессия в неподходящем состоянии")
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router?.popToRoot()
        case .notFound:
            showToast?("Сессия не найдена")
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router?.popToRoot()
        default:
            showToast?("Не удалось сохранить пин")
        }
    }
}

private extension String {
    var nonEmptyOrNil: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
