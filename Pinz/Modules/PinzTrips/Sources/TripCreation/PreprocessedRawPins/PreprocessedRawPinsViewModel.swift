import SwiftUI
import Foundation
import PinzNetworking
import PinzBase
import PinzDomain

@MainActor @Observable
final class PreprocessedRawPinsViewModel {

    enum Route {
        case back
        case review(tripId: String, pins: [Pin])
    }

    enum Intent {
        case navigate(Route)
        case deleteMedia(RawPinMedia, fromPin: String)
        case mergePins(firstIndex: Int, secondIndex: Int)
        case addPin
        case moveMedia(RawPinMedia, fromPin: Int, toPin: Int)
    }

    enum AsyncIntent {
        case `continue`
    }

    let tripId: String
    var pins: RawPins
    private(set) var isLoading = false

    private let networkService = NetworkService.shared
    private var router: AppRouting?
    private var deletedMediaIds: [String] = []

    init(tripId: String, pins: RawPins) {
        self.tripId = tripId
        self.pins = pins
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            case .review(let tripId, let pins):
                router?.navigateToTripCreationReview(tripId: tripId, pins: pins)
            }
        case let .deleteMedia(media, pinID):
            withAnimation(.easeInOut(duration: 0.3)) {
                if let pinIndex = pins.pins.firstIndex(where: { $0.id == pinID }) {
                    pins.pins[pinIndex].medias.removeAll { $0.id == media.id }
                    deletedMediaIds.append(media.id)
                }
            }
        case let .mergePins(firstIndex, secondIndex):
            guard firstIndex != secondIndex,
                  firstIndex < pins.pins.count,
                  secondIndex < pins.pins.count else { return }
            withAnimation(.easeInOut(duration: 0.3)) {
                pins.pins[firstIndex].medias += pins.pins[secondIndex].medias
                pins.pins.remove(at: secondIndex)
            }
        case .addPin:
            withAnimation(.easeInOut(duration: 0.3)) {
                let newId = UUID().uuidString
                pins.pins.append(RawPin(id: newId, medias: []))
            }
        case let .moveMedia(media, fromPin, toPin):
            withAnimation(.easeInOut(duration: 0.3)) {
                guard fromPin != toPin,
                      fromPin < pins.pins.count,
                      toPin < pins.pins.count else { return }
                pins.pins[fromPin].medias.removeAll { $0.id == media.id }
                pins.pins[toPin].medias.append(media)
            }
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            changeLoading(to: true)
            defer { changeLoading(to: false) }

            let draftPins = pins.pins.map { DraftPinInputDTO(draftPinId: $0.id, mediaIds: $0.medias.map(\.id)) }
            let waitTask = Task {
                try await networkService.waitForTripProcessingCompleted(
                    tripId: tripId,
                    timeout: 30
                )
            }

            do {
                try await networkService.applyGroupsAndProcess(
                    tripId: tripId,
                    draftPins: draftPins,
                    deletedMediaIds: deletedMediaIds
                )
            } catch {
                waitTask.cancel()
                throw error
            }

            let reviewResponse: GetTripReviewDTO

            do {
                try await waitTask.value
                reviewResponse = try await networkService.getTripReview(tripId: tripId)
            } catch TripReviewWaitError.timeout {
                reviewResponse = try await Self.waitForReviewAfterTimeout(
                    tripId: tripId,
                    networkService: networkService
                )
                let reviewStatus = Self.normalizedReviewStatus(reviewResponse.status)
                guard reviewStatus == "PROCESSING" || reviewStatus == "DRAFT_FINAL_REVIEW" else {
                    throw TripReviewWaitError.webSocket(message: "unexpected review status: \(reviewStatus)")
                }
            } catch {
                throw error
            }

            let pins = reviewResponse.pins.enumerated().map { index, dto in dto.toPin(index: index) }
            dispatch(.navigate(.review(tripId: tripId, pins: pins)))
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    private func changeLoading(to isLoading: Bool) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
        }
    }

    private static func waitForReviewAfterTimeout(
        tripId: String,
        networkService: NetworkService,
        attempts: Int = 6,
        interval: TimeInterval = 2.0
    ) async throws -> GetTripReviewDTO {
        var lastResponse: GetTripReviewDTO?
        for attempt in 0..<attempts {
            if attempt > 0 {
                try await Task.sleep(nanoseconds: UInt64(interval * 1_000_000_000))
            }
            let response = try await networkService.getTripReview(tripId: tripId)
            let status = normalizedReviewStatus(response.status)
            if status == "PROCESSING" || status == "DRAFT_FINAL_REVIEW" {
                return response
            }
            lastResponse = response
        }
        if let lastResponse {
            return lastResponse
        }
        throw TripReviewWaitError.timeout
    }

    private static func normalizedReviewStatus(_ status: String) -> String {
        status.uppercased().trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
