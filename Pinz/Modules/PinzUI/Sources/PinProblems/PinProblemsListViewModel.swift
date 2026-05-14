import SwiftUI
import PinzBase
import PinzDomain

@MainActor
@Observable
public final class PinProblemsListViewModel {

    public enum Route {
        case back
    }

    public enum Intent {
        case navigate(Route)
    }

    public struct ProblemPin: Hashable {
        public let pin: Pin
        public let pinIndex: Int
        public let issueText: String

        public var id: String {
            "\(pinIndex)-\(pin.serverId ?? pin.name)"
        }
    }

    let draftBinding: PinProblemsDraftBinding
    public var pins: [Pin]

    private var router: AppRouting?

    public var pinsWithIssues: [ProblemPin] {
        pins.enumerated().compactMap { index, pin in
            let issueText = pin.issueKinds
                .map(\.localizedTitle)
                .joined(separator: ", ")
            guard !issueText.isEmpty else { return nil }
            return ProblemPin(pin: pin, pinIndex: index, issueText: issueText)
        }
    }

    public init(draftBinding: PinProblemsDraftBinding, pins: [Pin]) {
        self.draftBinding = draftBinding
        self.pins = pins
    }

    public func dispatch(_ intent: Intent) {
        switch intent {
        case .navigate:
            router?.pop()
        }
    }

    public func navigateToPinInfo(at index: Int, router: AppRouting?) {
        let currentPinsWithIssues = pinsWithIssues
        guard index >= 0, index < currentPinsWithIssues.count else {
            return
        }
        let pinProblem = currentPinsWithIssues[index]

        router?.navigateToPinInfo(
            pin: pinProblem.pin,
            updateAction: PinUpdateAction { [weak self] updatedPin in
                withAnimation {
                    var fixedPin = updatedPin
                    fixedPin.issues = self?.normalizeIssues(for: updatedPin) ?? []
                    self?.pins[pinProblem.pinIndex] = fixedPin
                    self?.syncDraftToRouter()
                }
            },
            deleteAction: nil
        )
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
        guard let router else { return }

        switch draftBinding {
        case .tripCreation(let tripId):
            if let draftPins = router.tripCreationDraftPins(for: tripId) {
                pins = draftPins
            } else {
                router.setTripCreationDraftPins(pins, for: tripId)
            }
        case .pinUpload(let sessionId):
            if let draft = router.pinUploadReviewDraftPin(forSessionId: sessionId) {
                pins = [draft]
            } else if let first = pins.first {
                router.setPinUploadReviewDraftPin(first, forSessionId: sessionId)
            }
        case .addMediaReview(let sessionId):
            if let draftPins = router.addMediaReviewDraftPins(forSessionId: sessionId) {
                pins = draftPins
            } else if !pins.isEmpty {
                router.setAddMediaReviewDraftPins(pins, forSessionId: sessionId)
            }
        }
    }

    private func normalizeIssues(for pin: Pin) -> [String] {
        var result: [String] = []
        if pin.coordinates == nil {
            result.append(Pin.Issue.missingCoordinates.rawValue)
        }
        if pin.startDate == nil || pin.endDate == nil {
            result.append(Pin.Issue.missingDates.rawValue)
        }
        return result
    }

    private func syncDraftToRouter() {
        guard let router else { return }
        switch draftBinding {
        case .tripCreation(let tripId):
            router.setTripCreationDraftPins(pins, for: tripId)
        case .pinUpload(let sessionId):
            guard let pin = pins.first else { return }
            router.setPinUploadReviewDraftPin(pin, forSessionId: sessionId)
        case .addMediaReview(let sessionId):
            router.setAddMediaReviewDraftPins(pins, forSessionId: sessionId)
        }
    }
}

private extension Pin.Issue {
    var localizedTitle: String {
        switch self {
        case .missingCoordinates:
            PinzBaseStrings.TripCreationProblems.Issue.missingCoordinates
        case .missingDates:
            PinzBaseStrings.TripCreationProblems.Issue.missingDates
        }
    }
}
