import SwiftUI
import PinzDomain

public struct TagsPinShortInfo: View {
    let pin: Pin

    public init(pin: Pin) {
        self.pin = pin
    }

    public var body: some View {
        if !pin.tags.isEmpty {
            TagsView(tags: pin.tags, onTagAdd: {_ in }, onTagDelete: {_ in }, style: .default)
                .padding(.horizontal, 16)
        }
    }
}
