import SwiftUI

public struct BottomGradientWithButtons<Buttons: View>: View {

    let height: CGFloat
    @ViewBuilder var buttons: Buttons

    public init(
        height: CGFloat = 130,
        @ViewBuilder buttons: () -> Buttons,
    ) {
        self.height = height
        self.buttons = buttons()
    }

    public var body: some View {
        ZStack {
            VStack {
                Spacer()
                GradientView(style: .bottom, color: PinzUIAsset.background.swiftUIColor, height: height)
            }.ignoresSafeArea()

            VStack {
                Spacer()
                buttons
            }.padding(12)
        }
    }
}
