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

                LinearGradient(
                    gradient: Gradient(colors: [
                        PinzUIAsset.background.swiftUIColor,
                        Color.clear,
                    ]),
                    startPoint: .bottom,
                    endPoint: .top
                ).frame(height: 130)
            }.ignoresSafeArea()

            VStack {
                Spacer()
                buttons
            }.padding(12)
        }
    }
}
