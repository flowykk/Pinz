import SwiftUI

public struct Header<
    LeftContent: View,
    CenterContent: View,
    RightContent: View,
    AdditionalContent: View
>: View {
    private let backgroundColor: Color
    private let leftView: LeftContent
    private let centerView: CenterContent
    private let rightView: RightContent
    private let additionalView: AdditionalContent
    private let height: CGFloat?

    public init(
        backgroundColor: Color = Color.black,
        @ViewBuilder leftView: () -> LeftContent = { EmptyView() },
        @ViewBuilder centerView: () -> CenterContent = { EmptyView() },
        @ViewBuilder rightView: () -> RightContent = { EmptyView() },
        @ViewBuilder additionalView: () -> AdditionalContent = { EmptyView() },
        height: CGFloat? = 44
    ) {
        self.backgroundColor = backgroundColor
        self.leftView = leftView()
        self.centerView = centerView()
        self.rightView = rightView()
        self.additionalView = additionalView()
        self.height = height
    }

    public var body: some View {
        VStack {
            ZStack {
                HStack {
                    HStack(spacing: 0) { leftView }
                    Spacer()
                    HStack(spacing: 0) { rightView }
                }
                HStack {
                    Spacer()
                    centerView
                    Spacer()
                }
            }
            additionalView
                .padding(.bottom, 8)
        }
        .ifLet(height) { view, value in
            view.frame(height: height)
        }
        .padding(.horizontal, 8)
        .background(.clear)
    }
}
