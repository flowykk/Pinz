import SwiftUI
import PinzUI

/// Add-media trip review: list pin issues; user fixes pins on the review screen (tap row there).
public struct AddMediaProblemsView: View {

    private let sessionId: String

    public init(tripId: String, sessionId: String) {
        _ = tripId
        self.sessionId = sessionId
    }

    public var body: some View {
        PinProblemsListView(
            draftBinding: .addMediaReview(sessionId: sessionId),
            pins: [],
            rowTapBehavior: .none
        )
    }
}
