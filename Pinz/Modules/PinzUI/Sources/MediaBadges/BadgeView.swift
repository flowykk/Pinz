import SwiftUI

public struct BadgeView: View {

    public enum Icon: String {
        case lock = "lock.fill"
        case lockOpen = "lock.open.fill"
        case video = "video.fill"
        case trash = "trash.fill"
    }

    private let icon: Icon
    private let color: Color
    private let cornerRadius: CGFloat
    private let badgeSize: CGFloat
    private let iconSize: CGFloat

    public init(
        icon: Icon,
        color: Color = .white,
        cornerRadius: CGFloat = 10,
        badgeSize: CGFloat = 24,
        iconSize: CGFloat = 12
    ) {
        self.icon = icon
        self.color = color
        self.cornerRadius = cornerRadius
        self.badgeSize = badgeSize
        self.iconSize = iconSize
    }

    public var body: some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .fill(.ultraThinMaterial)
            .frame(badgeSize)
            .overlay {
                Image(systemName: icon.rawValue)
                    .roundedFount(size: iconSize, foregroundColor: color)
            }
    }
}
