import SwiftUI
import PinzBase
import PinzDomain

@MainActor @Observable
final class TripCreationProblemsViewModel {

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
    }

    struct ProblemPin: Hashable {
        let pin: Pin
        let pinIndex: Int
        let issueText: String

        var id: String {
            "\(pinIndex)-\(pin.serverId ?? pin.name)"
        }
    }

    let tripId: String
    var pins: [Pin]

    private var router: AppRouting?

    var pinsWithIssues: [ProblemPin] {
        pins.enumerated().compactMap { index, pin in
            let issueText = pin.issueKinds
                .map(\.localizedTitle)
                .joined(separator: ", ")
            guard !issueText.isEmpty else { return nil }
            return ProblemPin(pin: pin, pinIndex: index, issueText: issueText)
        }
    }

    init(tripId: String, pins: [Pin]) {
        self.tripId = tripId
        self.pins = pins
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .navigate:
            router?.pop()
        }
    }

    func navigateToPinInfo(at index: Int, router: AppRouting?) {
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
                    self?.syncDraftPins()
                }
            }
        )
    }

    func setRouter(_ router: AppRouting?) {
        self.router = router
        guard let router else { return }

        if let draftPins = router.tripCreationDraftPins(for: tripId) {
            pins = draftPins
        } else {
            router.setTripCreationDraftPins(pins, for: tripId)
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

    private func syncDraftPins() {
        guard let router else { return }
        router.setTripCreationDraftPins(pins, for: tripId)
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
