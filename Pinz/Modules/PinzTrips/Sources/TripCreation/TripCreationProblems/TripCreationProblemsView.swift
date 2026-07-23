import SwiftUI
import PinzDomain
import PinzUI

public struct TripCreationProblemsView: View {

    private let tripId: String
    private let pins: [Pin]

    public init(tripId: String, pins: [Pin]) {
        self.tripId = tripId
        self.pins = pins
    }

    public var body: some View {
        PinProblemsListView(draftBinding: .tripCreation(tripId: tripId), pins: pins)
    }
}
