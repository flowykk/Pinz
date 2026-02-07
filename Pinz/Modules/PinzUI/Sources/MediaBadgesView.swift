import SwiftUI
import PinzDomain

public struct MediaBadgesView: View {

    public enum Icon: String {
        case lock = "lock.fill"
        case lockOpen = "lock.open.fill"
        case video = "video.fill"
    }

    private let media: MediaItem

    public init(media: MediaItem) {
        self.media = media
    }

    public var body: some View {
        VStack {
            HStack {
                badgeItem(icon: media.isPrivate ? .lock : .lockOpen)
                Spacer()
                if media.type == .video {
                    badgeItem(icon: .video)
                }
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func badgeItem(icon: Icon) -> some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(.ultraThinMaterial)
            .frame(24)
            .overlay {
                Image(systemName: icon.rawValue)
                    .roundedFount(size: 12, foregroundColor: .white)
            }
    }
}
