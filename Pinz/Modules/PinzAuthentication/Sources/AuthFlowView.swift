import SwiftUI
import MapKit
import PinzUI

public struct AuthFlowView: View {
    @State private var viewModel = AuthFlowViewModel()
    @FocusState private var isFieldFocused: Bool
    
    public init() {}
    
    public var body: some View {
        Map(position: $viewModel.cameraPosition) { }
            .mapStyle(.imagery(elevation: .realistic))
            .mapControlVisibility(.hidden)
            .edgesIgnoringSafeArea(.all)
            .overlay {
                ZStack {
                    VStack {
                         if viewModel.state != .welcome {
                             PinzHeader(leftView: {
                                 PinzButton(type: .leftChevron, tint: .white) {
                                     viewModel.dispatch(.back)
                                 }
                             })
                         }

                        Spacer()

                        ZStack {
                            switch viewModel.state {
                            case .welcome:
                                welcomeOverlay.transition(.opacity)
                            case .email:
                                emailInputOverlay.transition(.move(edge: .bottom))
                            case .auth:
                                authPasswordOverlay.transition(.move(edge: .bottom))
                            case let .register(registerState):
                                switch registerState {
                                case .code:
                                    registerCodeOverlay.transition(.move(edge: .bottom))
                                case .password:
                                    registerPasswordOverlay.transition(.move(edge: .bottom))
                                case .repeatPassword:
                                    registerRepeatPasswordOverlay.transition(.move(edge: .bottom))
                                case .nickname:
                                    registerNicknameOverlay.transition(.move(edge: .bottom))
                                }
                            }
                        }
                        .if(viewModel.state != .welcome) { view in
                            view
                                .padding(.horizontal, 6)
                                .padding(.bottom, 8)
                                .background(backgroundTextFieldGradient)
                        }
                    }
                }
            }
            .animation(.easeInOut(duration: 0.2), value: viewModel.state)
            .onAppear {
                viewModel.dispatch(.startRotation)
            }
    }

    private var backgroundTextFieldGradient: some View {
        LinearGradient(
            gradient: Gradient(colors: [
                Color.black.opacity(0.8),
                Color.clear
            ]),
            startPoint: .bottom,
            endPoint: .top
        ).frame(height: 264)
    }
}

// MARK: Main flow views

extension AuthFlowView {
    private var welcomeOverlay: some View {
        ZStack {
            VStack {
                Spacer()

                Button(action: {
                    viewModel.dispatch(.proceedFromWelcome)
                }) {
                    Image(systemName: "arrow.right")
                        .font(.system(size: 28))
                        .foregroundColor(.black)
                        .background(
                            Rectangle()
                                .fill(.white)
                                .cornerRadius(16)
                                .frame(width: 78, height: 54)
                        )
                }
                .padding(.bottom, 50)
            }

            VStack {
                Spacer()

                Rectangle()
                    .fill(.black)
                    .frame(maxWidth: .infinity, maxHeight: 36)
            }
            .ignoresSafeArea(edges: .bottom)
        }
    }

    private var emailInputOverlay: some View {
        PinzTextField(
            label: "email:",
            style: .default(placeholder: "your@email.com"),
            text: $viewModel.text,
            keyboardType: .emailAddress,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
        .focused($isFieldFocused)
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }
}

// MARK: - Auth flow views

extension AuthFlowView {
    private var authPasswordOverlay: some View {
        PinzTextField(
            label: "password:",
            style: .default(placeholder: "shhhhhhhhh"),
            text: $viewModel.text,
            keyboardType: .default,
            action: .plain {
                print("Gone to Feed")
            }
        )
        .focused($isFieldFocused)
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }
}

// MARK: - Register flow views

extension AuthFlowView {
    private var registerCodeOverlay: some View {
        PinzTextField(
            label: "verification code:",
            style: .segmented(4),
            text: $viewModel.text,
            keyboardType: .default,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
    }

    private var registerPasswordOverlay: some View {
        PinzTextField(
            label: "password:",
            style: .default(placeholder: "shhhhhhhhh"),
            text: $viewModel.text,
            keyboardType: .default,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
        .focused($isFieldFocused)
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }

    private var registerRepeatPasswordOverlay: some View {
        PinzTextField(
            label: "password:",
            style: .default(placeholder: "repeat shh"),
            text: $viewModel.text,
            keyboardType: .default,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
        .focused($isFieldFocused)
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }

    private var registerNicknameOverlay: some View {
        PinzTextField(
            label: "nickname:",
            style: .default(placeholder: "cool guy"),
            text: $viewModel.text,
            keyboardType: .default,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
        .focused($isFieldFocused)
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }
}
