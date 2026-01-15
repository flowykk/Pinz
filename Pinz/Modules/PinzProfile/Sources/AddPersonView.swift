import SwiftUI
import PinzUI

public struct AddPersonView: View {
    
    public init() {}
    
    public var body: some View {
        VStack {
            Text("Add Person")
                .roundedFount(size: 24, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(PinzUIAsset.background.swiftUIColor)
    }
}
