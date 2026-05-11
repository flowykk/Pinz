import Foundation
import PinzBase
import PinzNetworking

/// Resume / fresh-start helper for the pin-upload flow.
///
/// `GET /trips/{id}` не возвращает активную pin-upload сессию, поэтому единственный
/// источник истины — `PinUploadSessionStorage`. Логика:
/// - Если в локальном хранилище есть `session_id`, дёргаем `pinUploadGetReview`
///   и решаем, на какой экран навигировать по `processing_status`.
/// - Если сессия не найдена на бэке (404) — чистим локально и стартуем заново.
/// - Если сессия есть, но статус неизвестный — то же, что 404.
/// - Если нет локального `session_id` — сразу на старт-экран.
public enum PinUploadEntryResolver {

    @MainActor
    public static func resume(
        tripId: String,
        networkService: NetworkServiceProtocol,
        router: AppRouting?,
        showToast: ((String) -> Void)? = nil
    ) async {
        guard let router else { return }

        guard let sessionId = PinUploadSessionStorage.shared.sessionId(forTripId: tripId) else {
            router.navigateToPinUploadStart(tripId: tripId)
            return
        }

        do {
            let response = try await networkService.pinUploadGetReview(
                tripId: tripId,
                sessionId: sessionId
            )
            switch response.processingStatus.uppercased() {
            case "UPLOADING", "PROCESSING":
                router.navigateToPinUploadProcessing(tripId: tripId, sessionId: sessionId)
            case "READY_FOR_REVIEW":
                router.navigateToPinUploadReview(tripId: tripId, sessionId: sessionId)
            default:
                // Неизвестный/финальный статус — сессия мертва или backend в странном состоянии.
                PinUploadSessionStorage.shared.clear(forTripId: tripId)
                router.navigateToPinUploadStart(tripId: tripId)
            }
        } catch let httpError as HTTPError where httpError == .notFound {
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router.navigateToPinUploadStart(tripId: tripId)
        } catch let httpError as HTTPError where httpError == .conflict {
            showToast?("Сессия создания пина занята на другом устройстве. Начинаем заново.")
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router.navigateToPinUploadStart(tripId: tripId)
        } catch {
            // 401 retry уже встроен в pinUploadGetReview через retryOnUnauthorized.
            // Прочие ошибки (сеть, 5xx) — оставляем сессию в storage и сообщаем пользователю.
            showToast?("Не удалось восстановить сессию создания пина")
        }
    }
}
