import SwiftUI

public struct CodeInputTextField: View {
    public struct Style {
        var segmentsCount: Int
        var background: UIColor
        var cornerRadius: CGFloat
        var fontSize: CGFloat
        var width: CGFloat?
        var height: CGFloat

        public init(
            segmentsCount: Int,
            background: UIColor,
            cornerRadius: CGFloat,
            fontSize: CGFloat,
            width: CGFloat? = nil,
            height: CGFloat
        ) {
            self.segmentsCount = segmentsCount
            self.background = background
            self.cornerRadius = cornerRadius
            self.fontSize = fontSize
            self.width = width
            self.height = height
        }
    }

    @Binding
    var code: [String]
    var style: Style
    var onCodeFilled: () -> Void

    @State
    private var focusedField: Int = -1

    public init(
        code: Binding<[String]>,
        style: Style,
        onCodeFilled: @escaping () -> Void
    ) {
        self._code = code
        self.style = style
        self.onCodeFilled = onCodeFilled
    }

    var joinedCode: String {
        code.joined()
    }

    public var body: some View {
        HStack(spacing: 4) {
            ForEach(0..<style.segmentsCount, id: \.self) { index in
                CodeDigitTextFieldAdapter(
                    code: $code,
                    focusedField: $focusedField,
                    background: style.background,
                    cornerRadius: style.cornerRadius,
                    fontSize: style.fontSize,
                    tag: index
                )
                .frame(
                    maxWidth: style.width == nil ? .infinity : style.width,
                    maxHeight: style.height
                )
            }
        }
        .onChange(of: code) { _, newValue in focusedField = newValue.count }
        .onChange(of: joinedCode.count == style.segmentsCount) { _, newValue in
            if newValue {
                onCodeFilled()
            }
        }
        .onAppear {
            focusedField = 0
        }
    }
}
