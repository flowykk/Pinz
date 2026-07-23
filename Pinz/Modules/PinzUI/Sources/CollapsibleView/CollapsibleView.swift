import SwiftUI

public struct CollapsibleView<Content: View>: View {

    let limitedHeight: CGFloat
    @ViewBuilder var content: Content

    @State var collapsed: Bool = false
    @State var viewHeight: CGFloat = 0

    public init(
        limitedHeight: CGFloat = 500,
        @ViewBuilder content: () -> Content = { EmptyView() },
    ) {
        self.limitedHeight = limitedHeight
        self.content = content()
    }

    public var body: some View {
        content
            .measureHeight(for: $viewHeight)
            .frame(height: collapsed ? limitedHeight : nil)
            .if(viewHeight - limitedHeight > 100) { view in
                view.overlay { badgeOverlay }
            }
    }

    public var badgeOverlay: some View {
        VStack {
            HStack {
                Spacer()
                BadgeView(
                    icon: collapsed ? .expand : .collapse,
                    badgeSize: 36,
                    iconSize: 18
                ) {
                    collapsed.toggle()
                }
            }
            Spacer()
        }.padding(8)
    }
}
