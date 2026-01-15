import SwiftUI

public struct PinzTextField: View {
    public enum Action {
        case plain(() -> Void)
        case async(() async throws -> Void)
    }

    public enum Style {
        case `default`(placeholder: String)
        case segmented(Int = 4)
    }

    let label: String?
    let style: Style
    @Binding var text: String
    var keyboardType: UIKeyboardType = .default
    @State var isLoading: Bool = false
    var action: Action

    public init(
        label: String? = nil,
        style: Style,
        text: Binding<String>,
        keyboardType: UIKeyboardType = .default,
        action: Action
    ) {
        self.label = label
        self.style = style
        self._text = text
        self.keyboardType = keyboardType
        self.action = action
    }
    
    public var body: some View {
        HStack(spacing: 4) {
            let background = UIColor.white.withAlphaComponent(0.64)
            let fontSize: CGFloat = 16
            let height: CGFloat = 50
            let cornerRadius: CGFloat = 14

            if let label = label {
                Text(label)
                    .font(.system(size: fontSize, weight: .semibold, design: .monospaced))
                    .foregroundColor(.white)
            }

            switch style {
            case let .default(placeholder):
                TextField(placeholder, text: $text)
                    .font(.system(size: fontSize, weight: .semibold, design: .monospaced))
                    .keyboardType(keyboardType)
                    .autocapitalization(.none)
                    .padding(.horizontal, 10)
                    .frame(height: height)
                    .background(RoundedRectangle(cornerRadius: cornerRadius).fill(Color(uiColor: background)))

                Button {
                    executeAction()
                } label: {
                    Group {
                        if isLoading {
                            ProgressView()
                        } else {
                            Image(systemName: "arrow.right")
                                .font(.system(size: 20))
                                .foregroundColor(.black)
                        }
                    }
                    .frame(width: 60, height: height)
                    .background(
                        RoundedRectangle(cornerRadius: cornerRadius).fill(.white)
                    )
                }
                .disabled(isLoading)
            case let .segmented(segmentsCount):
                CodeInputTextField(
                    code: codeBinding(segmentsCount),
                    style: .init(
                        segmentsCount: segmentsCount,
                        background: background,
                        cornerRadius: cornerRadius,
                        fontSize: fontSize,
                        height: height
                    ),
                    onCodeFilled: { executeAction() }
                )
            }
        }
    }

    private func codeBinding(_ segmentsCount: Int) -> Binding<[String]> {
        Binding(
            get: {
                let chars = Array(text.prefix(segmentsCount))
                let result = chars.map { String($0) } + Array(repeating: "", count: max(0, segmentsCount - chars.count))
                return Array(result.prefix(segmentsCount))
            },
            set: { newValue in
                let padded = newValue + Array(repeating: "", count: max(0, segmentsCount - newValue.count))
                text = Array(padded.prefix(segmentsCount)).joined()
            }
        )
    }

    private func executeAction() {
        switch action {
        case let .async(action):
            isLoading = true
            Task {
                defer { isLoading = false }
                try await action()
            }
        case let .plain(action):
            action()
        }
    }
}
