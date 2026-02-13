import SwiftUI
import PinzBase
import PinzDomain

public struct TripShortInfoView: View {

    private let trip: Trip
    private let onTripTapped: (Trip) -> Void

    @Environment(\.appRouter) private var router

    public init(
        trip: Trip,
        onTripTapped: @escaping (Trip) -> Void
    ) {
        self.trip = trip
        self.onTripTapped = onTripTapped
    }

    public var body: some View {
        Button {
            onTripTapped(trip)
        } label: {
            HStack() {
                Image(uiImage: trip.image ?? PinzDomainAsset.groupPlaceholder.image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: 100, height: 130)
                    .clipped()
                    .cornerRadius(16)

                VStack(alignment: .leading, spacing: 4) {
                    Text(trip.name)
                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Text("\(trip.category.value) / \(trip.season.value)")
                        .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    HStack(spacing: 10) {
                        StatisticView(icon: "person.2.fill", text: String(trip.members.count))
                        StatisticView(icon: "photo.stack.fill", text: "100")
                        StatisticView(icon: "calendar", text: "12 days")
                    }
                    if let description = trip.description {
                        Text(description)
                            .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                            .lineLimit(5)
                    }
                }

                Image(systemName: "chevron.right")
                    .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
        }
        .buttonStyle(.plain)
        .frame(height: 130)
    }
}
