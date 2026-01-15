import SwiftUI

public struct Header<LeftContent: View, CenterContent: View, RightContent: View>: View {
    private let backgroundColor: Color
    private let leftView: LeftContent
    private let centerView: CenterContent
    private let rightView: RightContent

    public init(
        backgroundColor: Color = Color.black,
        @ViewBuilder leftView: () -> LeftContent = { EmptyView() },
        @ViewBuilder centerView: () -> CenterContent = { EmptyView() },
        @ViewBuilder rightView: () -> RightContent = { EmptyView() }
    ) {
        self.backgroundColor = backgroundColor
        self.leftView = leftView()
        self.centerView = centerView()
        self.rightView = rightView()
    }

    public var body: some View {
        ZStack {
            HStack {
                leftView
                Spacer()
                rightView
            }
            HStack {
                Spacer()
                centerView
                Spacer()
            }
        }
        .frame(height: 44)
        .padding(.horizontal, 8)
        .background(.clear)
    }
}
