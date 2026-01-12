import SwiftUI
import UIKit

public struct CodeDigitTextFieldAdapter: UIViewRepresentable {
    @Binding var code: [String]
    @Binding var focusedField: Int
    var background: UIColor
    var cornerRadius: CGFloat
    var fontSize: CGFloat
    var index: Int

    public init(
        code: Binding<[String]>,
        focusedField: Binding<Int>,
        background: UIColor,
        cornerRadius: CGFloat,
        fontSize: CGFloat,
        tag: Int
    ) {
        self._code = code
        self._focusedField = focusedField
        self.background = background
        self.cornerRadius = cornerRadius
        self.fontSize = fontSize
        self.index = tag
    }

    public func makeUIView(context: Context) -> CodeDigitTextField {
        let textField = CodeDigitTextField()
        textField.textAlignment = .center
        textField.keyboardType = .numberPad
        textField.font = UIFont.monospacedSystemFont(ofSize: fontSize, weight: .semibold)
        textField.backgroundColor = background
        textField.layer.cornerRadius = cornerRadius
        textField.delegate = context.coordinator
        textField.addTarget(context.coordinator, action: #selector(Coordinator.textChanged), for: .editingChanged)
        textField.onBackspace = { isEmpty in
            if isEmpty {
                context.coordinator.stepBackward()
            }
            let targetIndex = isEmpty ? max(0, index - 1) : index
            if targetIndex < code.count {
                code[targetIndex] = ""
            }
        }
        return textField
    }

    public func updateUIView(_ uiView: CodeDigitTextField, context _: Context) {
        uiView.text = index < code.count ? code[index] : ""
        if focusedField == index, !uiView.isFirstResponder {
            DispatchQueue.main.async {
                uiView.becomeFirstResponder()
            }
        }
    }

    public func makeCoordinator() -> Coordinator {
        Coordinator(code: $code, focusedField: $focusedField, tag: index)
    }

    public final class Coordinator: NSObject, UITextFieldDelegate {
        @Binding var code: [String]
        @Binding var focusedField: Int
        var tag: Int

        init(code: Binding<[String]>, focusedField: Binding<Int>, tag: Int) {
            _code = code
            _focusedField = focusedField
            self.tag = tag
        }

        @objc func textChanged(_ textField: UITextField) {
            if let textValue = textField.text, !textValue.isEmpty, tag < code.count {
                code[tag] = String(textValue.prefix(1))
                stepForward()
            }
        }

        public func textField(
            _: UITextField,
            shouldChangeCharactersIn _: NSRange,
            replacementString string: String
        ) -> Bool {
            guard tag < code.count else { return false }
            
            // No need to move cursor, when backspacing on current cell
            if string.isEmpty, !code[tag].isEmpty {
                return true
            }

            // Step forward, when replacing non-empty current cell with new value
            if !string.isEmpty, !code[tag].isEmpty {
                code[tag] = string
                stepForward()
                return false
            }

            // Step backward, when current cell is empty
            if string.isEmpty, code[tag].isEmpty {
                stepBackward()
                return false
            }

            return true
        }

        func stepForward() {
            focusedField = tag + 1
        }

        func stepBackward() {
            focusedField = max(0, tag - 1)
        }
    }
}

public final class CodeDigitTextField: UITextField {
    var onBackspace: ((Bool) -> Void)?

    public override func deleteBackward() {
        if let text {
            onBackspace?(text.isEmpty)
        }
        super.deleteBackward()
    }
}
