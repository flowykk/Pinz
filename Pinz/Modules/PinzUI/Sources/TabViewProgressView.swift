import SwiftUI

struct TabViewProgressView: View {

    let numberOfPages: Int
    let currentIndex: Int

    private let circleSize: CGFloat = 8
    private let circleSpacing: CGFloat = 4

    private let primaryColor = PinzUIAsset.textSecondary.swiftUIColor
    private let secondaryColor = PinzUIAsset.textSecondary.swiftUIColor.opacity(0.6)

    private let smallScale: CGFloat = 0.6

    var body: some View {
        HStack(spacing: circleSpacing) {
            ForEach(0..<numberOfPages) { index in
                if shouldShowIndex(index) {
                    Circle()
                        .fill(currentIndex == index ? primaryColor : secondaryColor)
                        .scaleEffect(currentIndex == index ? 1 : smallScale)
                        .frame(width: circleSize, height: circleSize)
                        .transition(AnyTransition.opacity.combined(with: .scale))
                        .id(index)
                }
            }
        }
    }

    private func shouldShowIndex(_ index: Int) -> Bool {
        ((currentIndex - 2)...(currentIndex + 2)).contains(index)
    }
}
