import SwiftUI

public struct PinzTextField: View {
    let label: String?
    let placeholder: String
    @Binding var text: String
    var keyboardType: UIKeyboardType = .default
    var autocapitalization: TextInputAutocapitalization = .never
    var action: (() -> Void)?
    
    public init(
        label: String? = nil,
        placeholder: String,
        text: Binding<String>,
        keyboardType: UIKeyboardType = .default,
        autocapitalization: TextInputAutocapitalization = .never,
        action: (() -> Void)? = nil
    ) {
        self.label = label
        self.placeholder = placeholder
        self._text = text
        self.keyboardType = keyboardType
        self.autocapitalization = autocapitalization
        self.action = action
    }
    
    public var body: some View {
        HStack(spacing: 4) {
            if let label = label {
                Text(label)
                    .font(.system(size: 16, weight: .semibold, design: .monospaced))
                    .foregroundColor(.white)
            }
            
            TextField(placeholder, text: $text)
                .textFieldStyle(.plain)
                .font(.system(size: 16, weight: .semibold, design: .monospaced))
                .keyboardType(keyboardType)
                .padding(.horizontal, 10)
                .frame(height: 50)
                .background(
                    RoundedRectangle(cornerRadius: 14)
                        .fill(Color.white.opacity(0.64))
                )

            if let action = action {
                Button(action: action) {
                    Image(systemName: "arrow.right")
                        .font(.system(size: 20))
                        .foregroundColor(.black)
                        .frame(width: 64, height: 50)
                        .background(
                            RoundedRectangle(cornerRadius: 14)
                                .fill(.white)
                        )
                }
            } else {
                Image(systemName: "arrow.right")
                    .font(.system(size: 20))
                    .foregroundColor(.black)
                    .frame(width: 64, height: 50)
                    .background(
                        RoundedRectangle(cornerRadius: 14)
                            .fill(.white)
                    )
            }
        }
    }
}

#Preview {
    VStack(spacing: 20) {
        PinzTextField(
            label: "email:",
            placeholder: "Enter your email",
            text: .constant("test@example.com"),
            keyboardType: .emailAddress,
            action: {
                print("Action triggered!")
            }
        )
        
        PinzTextField(
            placeholder: "Without action",
            text: .constant("")
        )
    }
    .padding()
    .background(Color.black)
}

