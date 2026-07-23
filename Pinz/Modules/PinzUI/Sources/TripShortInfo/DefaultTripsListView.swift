import SwiftUI
import PinzDomain

public struct DefaultTripsListView: View {

    private let trips: [Trip]
    private let onTripTapped: (Trip) -> Void

    public init(trips: [Trip], onTripTapped: @escaping (Trip) -> Void = { _ in }) {
        self.trips = trips
        self.onTripTapped = onTripTapped
    }

    public var body: some View {
        VStack {
            ForEach(trips.indices, id: \.self) { index in
                TripShortInfoView(trip: trips[index], onTripTapped: onTripTapped)
                    .padding(.horizontal, 12)
                if index != trips.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
    }
}
