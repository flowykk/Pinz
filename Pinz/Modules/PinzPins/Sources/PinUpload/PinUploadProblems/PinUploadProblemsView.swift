import SwiftUI
import PinzUI

/// Pin upload review: same UX as trip creation problems, backed by [`PinProblemsListView`](PinProblemsListView).
public struct PinUploadProblemsView: View {

    private let sessionId: String

    public init(tripId: String, sessionId: String, targetPinId: String?) {
        _ = tripId
        _ = targetPinId
        self.sessionId = sessionId
    }

    public var body: some View {
        PinProblemsListView(
            draftBinding: .pinUpload(sessionId: sessionId),
            pins: [],
            rowTapBehavior: .none
        )
    }
}
