import SwiftUI
import PinzUI

public struct AppIconView: View {

    @State private var viewModel: AppIconViewModel

    public init(viewModel: AppIconViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        icon.if(viewModel.selected) { view in
            view.overlay(
                RoundedRectangle(cornerRadius: 16)
                    .strokeBorder(PinzUIAsset.accentGreen.swiftUIColor, lineWidth: 3)
            )
        }
    }

    public var icon: some View {
        Image(viewModel.name)
            .resizable()
            .aspectRatio(contentMode: .fit)
            .frame(70)
            .cornerRadius(16)
    }
}
