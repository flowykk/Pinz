import SwiftUI
import PinzDomain

public struct DefaultTripsListView: View {

    private let trips: [Trip]

    public init(trips: [Trip]) {
        self.trips = trips
    }

    public var body: some View {
        VStack {
            ForEach(trips.indices, id: \.self) { index in
                TripShortInfoView(trip: trips[index]) { _ in }
                    .padding(.horizontal, 12)
                if index != trips.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
    }
}
