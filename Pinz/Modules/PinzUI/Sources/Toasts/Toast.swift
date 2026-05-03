import SwiftUI

public struct ToastView: View {
    @State
    private var controller: ToastController
    private var title: String

    public init(controller: ToastController) {
        self.controller = controller
        title = controller.state.title
    }

    public var body: some View {
        VStack(spacing: 0) {
            if controller.state.isPresented {
                content
                    .padding(.horizontal, 12)
                    .transition(.move(edge: .top).combined(with: .opacity))
                    .onTapGesture {
                        controller.hide()
                    }
            }

            Spacer()
        }
    }

    private var content: some View {
        HStack(alignment: .center, spacing: 8) {
            Text(title)
                .roundedFont(size: 14, foregroundColor: PinzUIAsset.textPrimaryInverted.swiftUIColor)
                .padding(.vertical)

            Spacer()
        }
        .padding(.leading)
        .padding(.trailing)
        .background(PinzUIAsset.backgroundInverted.swiftUIColor)
        .cornerRadius(20)
    }
}
