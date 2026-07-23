import SwiftUI
import PinzUI
import PinzBase

struct UnselectedTripView: View {
    var body: some View {
        VStack() {
            Group {
                Text(PinzBaseStrings.Trips.Empty.noSelection)
                let parts = PinzBaseStrings.Trips.Empty.noSelectionHint.components(separatedBy: "[chevron.down]")
                Text(parts[0]) + Text(Image(systemName: "chevron.down")) + Text(parts.count > 1 ? parts[1] : "")
            }.multilineTextAlignment(.center)
        }
        .roundedFont(size: 18, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
        .padding(.horizontal, 12)
    }
}
