import SwiftUI

public struct PinzTextField: View {
    public enum Action {
        case plain(() -> Void)
        case async(() async throws -> Void)
    }

    let label: String?
    let placeholder: String
    @Binding var text: String
    var keyboardType: UIKeyboardType = .default
    var autocapitalization: TextInputAutocapitalization = .never
    @State var isLoading: Bool = false
    var action: Action

    public init(
        label: String? = nil,
        placeholder: String,
        text: Binding<String>,
        keyboardType: UIKeyboardType = .default,
        autocapitalization: TextInputAutocapitalization = .never,
        action: Action
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

            Button {
                switch action {
                case .async(let action):
                    isLoading = true
                    Task {
                        defer { isLoading = false }
                        try await action()
                    }
                case .plain(let action):
                    action()
                }
            } label: {
                Group {
                    if isLoading {
                        ProgressView()
                            .tint(.black)
                    } else {
                        Image(systemName: "arrow.right")
                            .font(.system(size: 20))
                            .foregroundColor(.black)
                    }
                }
                .frame(width: 60, height: 50)
                .background(
                    RoundedRectangle(cornerRadius: 14)
                        .fill(.white)
                )
            }
            .disabled(isLoading)
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
            action: .plain {
                print("Action triggered!")
            }
        )
        
        PinzTextField(
            placeholder: "Without action",
            text: .constant(""),
            action: .plain { }
        )
    }
    .padding()
    .background(Color.black)
}

