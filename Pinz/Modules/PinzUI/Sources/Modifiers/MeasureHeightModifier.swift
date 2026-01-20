import SwiftUI

public struct MeasureHeightModifier: ViewModifier {
    @Binding private var height: CGFloat

    public init(height: Binding<CGFloat>) {
        self._height = height
    }

    public func body(content: Content) -> some View {
        content
            .background(
                GeometryReader { geometry in
                    Color.clear
                        .onAppear {
                            height = geometry.size.height
                        }
                }
            )
    }
}

extension View {
    func measureHeight(for height: Binding<CGFloat>) -> some View {
        self.modifier(MeasureHeightModifier(height: height))
    }
}
