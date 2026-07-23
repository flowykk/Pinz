import SwiftUI
import PinzDomain

public struct DefaultPinsListView: View {

    private let pins: [Pin]
    private let hideTags: Bool
    private let hideMediaBadges: Bool
    private let dismissBeforeMediaInfo: Bool
    private let allowsMediaPrivacyChange: Bool
    private let pinTapped: (Pin) -> Void
    private let onMediaUpdated: ((MediaItem, Pin) -> Void)?

    public init(
        pins: [Pin],
        hideTags: Bool = false,
        hideMediaBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        allowsMediaPrivacyChange: Bool = true,
        pinTapped: @escaping (Pin) -> Void,
        onMediaUpdated: ((MediaItem, Pin) -> Void)? = nil
    ) {
        self.pins = pins
        self.hideTags = hideTags
        self.hideMediaBadges = hideMediaBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.allowsMediaPrivacyChange = allowsMediaPrivacyChange
        self.pinTapped = pinTapped
        self.onMediaUpdated = onMediaUpdated
    }

    public var body: some View {
        VStack {
            ForEach(pins.indices, id: \.self) { index in
                DefaultPinShortInfoView(
                    pin: pins[index],
                    hideTags: hideTags,
                    hideMediaBadges: hideMediaBadges,
                    dismissBeforeMediaInfo: dismissBeforeMediaInfo,
                    allowsMediaPrivacyChange: allowsMediaPrivacyChange,
                    pinTapped: pinTapped,
                    onMediaUpdated: onMediaUpdated.map { handler in { media in handler(media, pins[index]) } }
                )
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
    }
}
