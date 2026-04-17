import SwiftUI
import PinzUI
import PinzBase

struct NoPinsPlaceholderView: View {
    let fillAvailableSpace: Bool

    init(fillAvailableSpace: Bool = true) {
        self.fillAvailableSpace = fillAvailableSpace
    }

    var body: some View {
        Text(PinzBaseStrings.TripPins.Empty.noPins)
            .roundedFont(size: 18, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            .multilineTextAlignment(.center)
            .padding(.horizontal, 12)
            .frame(maxWidth: .infinity, alignment: .center)
            .frame(maxHeight: fillAvailableSpace ? .infinity : nil, alignment: .center)
    }
}
