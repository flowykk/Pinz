import SwiftUI

public enum GradientStyle {
    case bottom
    case top
}

public struct GradientView: View {
    
    private let style: GradientStyle
    private let color: Color
    private let opacity: CGFloat
    private let height: CGFloat
    private let needsBlur: Bool

    public init(
        style: GradientStyle = .bottom,
        color: Color = .black,
        opacity: CGFloat = 0.6,
        height: CGFloat,
        needsBlur: Bool = true
    ) {
        self.style = style
        self.color = color
        self.opacity = opacity
        self.height = height
        self.needsBlur = needsBlur
    }
    
    public var body: some View {
        LinearGradient(
            gradient: Gradient(colors: gradientColors),
            startPoint: .bottom,
            endPoint: .top
        )
        .if(needsBlur) { view in
            view.background(alignment: style == .bottom ? .bottom : .top) {
                ProgressiveBlurView(
                    maxBlurRadius: 5,
                    direction: style == .bottom ? .blurredBottomClearTop : .blurredTopClearBottom
                )
                .ignoresSafeArea()
            }
        }
        .frame(height: height)
    }
    
    private var gradientColors: [Color] {
        switch style {
        case .bottom: return [color.opacity(opacity), .clear]
        case .top: return [.clear, color.opacity(opacity)]
        }
    }
}
