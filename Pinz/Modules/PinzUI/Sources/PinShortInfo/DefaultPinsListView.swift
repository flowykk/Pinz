import SwiftUI
import PinzDomain

public struct DefaultPinsListView: View {

    private let pins: [Pin]
    private let hideTags: Bool
    private let hideMediaBadges: Bool
    private let dismissBeforeMediaInfo: Bool
    private let pinTapped: (Pin) -> Void

    public init(
        pins: [Pin],
        hideTags: Bool = false,
        hideMediaBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        pinTapped: @escaping (Pin) -> Void
    ) {
        self.pins = pins
        self.hideTags = hideTags
        self.hideMediaBadges = hideMediaBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.pinTapped = pinTapped
    }

    public var body: some View {
        VStack {
            ForEach(pins.indices, id: \.self) { index in
                DefaultPinShortInfoView(
                    pin: pins[index],
                    hideTags: hideTags,
                    hideMediaBadges: hideMediaBadges,
                    dismissBeforeMediaInfo: dismissBeforeMediaInfo,
                    pinTapped: pinTapped,
                )
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
    }
}
