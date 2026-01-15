import SwiftUI
import PinzUI

public struct AppIconsGridView: View {

    @State private var viewModel: AppIconsViewModel = AppIconsViewModel()

    public init() {}

    public var body: some View {
        ScrollView(.horizontal) {
            HStack(spacing: 0) {
                ForEach(viewModel.appIcons) { icon in
                    AppIconView(viewModel: icon)
                        .onTapGesture {
                            viewModel.dispatch(.change(icon: icon))
                        }
                }
                .padding(.horizontal, 10)
            }
        }
        .scrollIndicators(.hidden)
        .padding(.vertical, 10)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(20)
    }
}
