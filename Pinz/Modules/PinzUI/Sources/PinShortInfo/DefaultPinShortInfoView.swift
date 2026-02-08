import SwiftUI
import PinzDomain

public struct DefaultPinShortInfoView: View {

    private let pin: Pin
    private let hideTags: Bool
    private let hideMediaBadges: Bool
    private let dismissBeforeMediaInfo: Bool
    private let pinTapped: (Pin) -> Void

    @Environment(\.dismiss) private var dismiss

    public init(
        pin: Pin,
        hideTags: Bool = false,
        hideMediaBadges: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        pinTapped: @escaping (Pin) -> Void,
    ) {
        self.pin = pin
        self.hideTags = hideTags
        self.hideMediaBadges = hideMediaBadges
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.pinTapped = pinTapped
    }

    public var body: some View {
        Button {
            pinTapped(pin)
        } label: {
            VStack(spacing: 0) {
                header
                medias.padding(.top, 6)
                if !hideTags {
                    TagsPinShortInfo(pin: pin).padding(.top, 2)
                }
            }
        }.buttonStyle(.plain)
    }

    var header: some View {
        HeaderPinShortInfo(pin: pin)
    }

    var medias: some View {
        MediasPinShortInfo(
            pin: pin,
            maxMedias: 15,
            hideMediaBadges: hideMediaBadges,
            dismissBeforeMediaInfo: dismissBeforeMediaInfo
        )
    }
}
