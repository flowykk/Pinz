import SwiftUI
import PinzUI

public struct AppIconView: View {

    @State private var viewModel: AppIconViewModel

    public init(viewModel: AppIconViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ZStack {
            icon

            if viewModel.selected {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundColor(.white)
                    .background(Circle().fill(Color(.systemGreen)).frame(width: 24, height: 24))
                    .fontWeight(.bold)
                    .offset(x: 23, y: -23)
            }
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
