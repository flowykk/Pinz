import Foundation
import PinzBase
import PinzNetworking

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
                PinUploadSessionStorage.shared.clear(forTripId: tripId)
                router.navigateToPinUploadStart(tripId: tripId)
            }
        } catch let httpError as HTTPError where httpError == .notFound {
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router.navigateToPinUploadStart(tripId: tripId)
        } catch let httpError as HTTPError where httpError == .conflict {
            showToast?(PinzBaseStrings.PinUpload.Creation.Resume.conflictSession)
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            router.navigateToPinUploadStart(tripId: tripId)
        } catch {
            showToast?(PinzBaseStrings.PinUpload.Creation.Resume.restoreFailed)
        }
    }

    @MainActor
    public static func resumeAddition(
        tripId: String,
        pinId rawPinId: String,
        networkService: NetworkServiceProtocol,
        router: AppRouting?,
        showToast: ((String) -> Void)? = nil
    ) async {
        guard let router else { return }

        let pinId = rawPinId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !pinId.isEmpty else { return }

        guard let sessionId = PinUploadAdditionSessionStorage.shared.sessionId(tripId: tripId, pinId: pinId) else {
            router.navigateToPinUploadStart(tripId: tripId, targetPinId: pinId)
            return
        }

        do {
            let response = try await networkService.pinUploadGetReview(
                tripId: tripId,
                sessionId: sessionId
            )
            switch response.processingStatus.uppercased() {
            case "UPLOADING", "PROCESSING":
                router.navigateToPinUploadProcessing(tripId: tripId, sessionId: sessionId, targetPinId: pinId)
            case "READY_FOR_REVIEW":
                router.navigateToPinUploadReview(tripId: tripId, sessionId: sessionId, targetPinId: pinId)
            default:
                PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
                router.navigateToPinUploadStart(tripId: tripId, targetPinId: pinId)
            }
        } catch let httpError as HTTPError where httpError == .notFound {
            PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
            router.navigateToPinUploadStart(tripId: tripId, targetPinId: pinId)
        } catch let httpError as HTTPError where httpError == .conflict {
            showToast?(PinzBaseStrings.PinUpload.Addition.Resume.conflictSession)
            PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
            router.navigateToPinUploadStart(tripId: tripId, targetPinId: pinId)
        } catch {
            showToast?(PinzBaseStrings.PinUpload.Addition.Resume.restoreFailed)
        }
    }
}
