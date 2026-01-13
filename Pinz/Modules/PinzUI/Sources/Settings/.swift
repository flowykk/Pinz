import SwiftUI

public struct SettingsGroupView: View {

    let settings: [Setting]

    public init(settings: [Setting]) {
        self.settings = settings
    }

    public var body: some View {
        VStack(spacing: 0) {
            ForEach(settings) { setting in
                setting.view
            }
            .padding(.horizontal, 10)
            .frame(height: 44)
        }
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(22)
    }
}

#Preview {
    SettingsGroupView(settings: [
        .default(.init(
            title: "title",
            value: "value",
            icon: "circle.fill",
            style: .default,
            action: .plain { print("smth") }
        )),
        .default(.init(
            title: "title1",
            value: "value2",
            icon: "circle3.fill",
            style: .default,
            action: .plain { print("smth") }
        ))
    ])
}
