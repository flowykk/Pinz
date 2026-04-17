import SwiftUI
import PinzBase
import PinzDomain

public struct TripShortInfoView: View {

    enum Constants {
        static let elementHeight: CGFloat = 125
        static let imageWidth: CGFloat = 100
    }

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

    private var daysText: String {
        guard let startDate = trip.startDate, let endDate = trip.endDate else {
            return "? дней"
        }
        let days = Calendar.current.dateComponents([.day], from: startDate, to: endDate).day ?? 0
        return "\(days) \(days.pluralizeDays())"
    }

    public var body: some View {
        Button {
            onTripTapped(trip)
        } label: {
            HStack(spacing: 12) {
                tripImage
                    .frame(width: Constants.imageWidth, height: Constants.elementHeight)
                    .cornerRadius(16)

                VStack(alignment: .leading, spacing: 4) {
                    Text(trip.name)
                        .roundedFont(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Text("\(trip.category.value) / \(trip.season.value)")
                        .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    HStack(spacing: 10) {
                        StatisticView(icon: "person.2.fill", text: String(trip.participantsCount))
                        StatisticView(icon: "photo.stack.fill", text: String(trip.mediaCount))
                        StatisticView(icon: "calendar", text: daysText)
                    }
                    if let description = trip.description {
                        Text(description)
                            .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                            .lineLimit(5)
                    }
                    Spacer(minLength: 0)
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
        }
        .buttonStyle(.plain)
        .frame(height: Constants.elementHeight)
    }

    @ViewBuilder
    private var tripImage: some View {
        if let coverUrl = trip.coverUrl, !coverUrl.isEmpty, let url = URL(string: coverUrl) {
            LoadableImageThumbnail(url: url) { state in
                remoteTripImage(for: state)
            }
        } else if let localImage = trip.image {
            image(for: localImage)
        } else {
            image(for: PinzDomainAsset.groupPlaceholder.image)
        }
    }

    @ViewBuilder
    private func remoteTripImage(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.2))
                .overlay {
                    ProgressView().tint(.white)
                }
        case .ready(let loadedImage):
            image(for: loadedImage)
        case .failure:
            if let localImage = trip.image {
                image(for: localImage)
            } else {
                image(for: PinzDomainAsset.groupPlaceholder.image)
            }
        }
    }

    private func image(for uiImage: UIImage) -> some View {
        Image(uiImage: uiImage)
            .resizable()
            .scaledToFill()
            .frame(width: Constants.imageWidth, height: Constants.elementHeight)
            .clipped()
    }
}
