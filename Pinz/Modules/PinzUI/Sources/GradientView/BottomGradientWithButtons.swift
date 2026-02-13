import SwiftUI

public struct BottomGradientWithButtons<Buttons: View>: View {

    @ViewBuilder var buttons: Buttons

    public init(
        @ViewBuilder buttons: () -> Buttons,
    ) {
        self.buttons = buttons()
    }

    public var body: some View {
        ZStack {
            VStack {
                Spacer()
                GradientView(style: .bottom, color: PinzUIAsset.background.swiftUIColor, opacity: 1.0, height: 130)
            }.ignoresSafeArea()

            VStack {
                Spacer()
                buttons
            }.padding(12)
        }
    }
}
