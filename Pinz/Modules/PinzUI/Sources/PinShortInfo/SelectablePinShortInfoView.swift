import SwiftUI
import PinzDomain

public struct SelectablePinShortInfoView: View {

    private let pin: Pin
    private let hideTags: Bool
    private let dismissBeforeMediaInfo: Bool
    private let isSelected: Bool
    private let onSelect: (() -> Void)?
    private let pinTapped: (Pin) -> Void

    @Environment(\.dismiss) private var dismiss

    public init(
        pin: Pin,
        hideTags: Bool = false,
        dismissBeforeMediaInfo: Bool = false,
        isSelected: Bool = false,
        onSelect: (() -> Void)? = nil,
        pinTapped: @escaping (Pin) -> Void,
    ) {
        self.pin = pin
        self.hideTags = hideTags
        self.dismissBeforeMediaInfo = dismissBeforeMediaInfo
        self.isSelected = isSelected
        self.onSelect = onSelect
        self.pinTapped = pinTapped
    }

    public var body: some View {
        VStack(spacing: 0) {
            header

            Button {
                pinTapped(pin)
            } label: {
                medias.padding(.top, 6)
            }.buttonStyle(.plain)

            if !hideTags {
                TagsPinShortInfo(pin: pin).padding(.top, 2)
            }
        }.opacity(pin.isPrivate ? 0.7 : 1)
    }

    var header: some View {
        HeaderPinShortInfo(
            pin: pin,
            selectable: true,
            isSelected: isSelected,
            onSelect: onSelect
        )
    }

    var medias: some View {
        MediasPinShortInfo(
            pin: pin,
            maxMedias: 15,
            selectable: true,
            dismissBeforeMediaInfo: dismissBeforeMediaInfo
        )
    }
}
