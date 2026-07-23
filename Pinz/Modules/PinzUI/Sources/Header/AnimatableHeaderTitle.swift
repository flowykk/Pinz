import SwiftUI

public struct AnimatableHeaderTitle: View {

    private let animatableTitle: String
    @Binding private var title: String

    public init(
        animatableTitle: String,
        title: Binding<String>
    ) {
        self.animatableTitle = animatableTitle
        self._title = title
    }

    public var body: some View {
        let isTitleEmpty = title.isEmpty
        let primaryColor = PinzUIAsset.textPrimary.swiftUIColor
        let secondaryColor = PinzUIAsset.textSecondary.swiftUIColor
        
        VStack(spacing: 0) {
            if !isTitleEmpty {
                Text(title)
                    .roundedFont(
                        size: 16,
                        weight: .semibold,
                        foregroundColor: PinzUIAsset.textPrimary.swiftUIColor
                    )
            }
            Text(animatableTitle)
                .roundedFont(
                    size: isTitleEmpty ? 16 : 14,
                    weight: isTitleEmpty ? .semibold : .medium,
                    foregroundColor: isTitleEmpty ? primaryColor : secondaryColor
                )
        }.animation(.default, value: title)
    }
}
