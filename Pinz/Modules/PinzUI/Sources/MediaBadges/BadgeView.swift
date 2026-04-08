import SwiftUI

public struct BadgeView: View {

    public enum Icon: String {
        case lock = "lock.fill"
        case lockOpen = "lock.open.fill"
        
        case video = "video.fill"
        
        case trash = "trash.fill"
        
        case expand = "arrow.down.backward.and.arrow.up.forward"
        case collapse = "arrow.up.forward.and.arrow.down.backward"

        case soundOn = "speaker.wave.2.fill"
        case soundOff = "speaker.slash.fill"
    }

    private let icon: Icon
    private let color: Color
    private let cornerRadius: CGFloat
    private let badgeSize: CGFloat
    private let iconSize: CGFloat
    private let action: (() -> Void)?

    public init(
        icon: Icon,
        color: Color = .white,
        cornerRadius: CGFloat = 10,
        badgeSize: CGFloat = 24,
        iconSize: CGFloat = 12,
        action: (() -> Void)? = nil
    ) {
        self.icon = icon
        self.color = color
        self.cornerRadius = cornerRadius
        self.badgeSize = badgeSize
        self.iconSize = iconSize
        self.action = action
    }

    public var body: some View {
        if let action {
            Button {
                withAnimation(.easeInOut(duration: 0.3)) {
                    action()
                }
            } label: {
                badge
            }.buttonStyle(.plain)
        } else {
            badge
        }

    }

    private var badge: some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .fill(.ultraThinMaterial)
            .frame(badgeSize)
            .overlay {
                Image(systemName: icon.rawValue)
                    .roundedFont(size: iconSize, foregroundColor: color)
            }
    }
}
