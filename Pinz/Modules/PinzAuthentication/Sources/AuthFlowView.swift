import SwiftUI
import MapKit
import PinzBase
import PinzUI

public struct AuthFlowView: View {
    @State private var viewModel = AuthFlowViewModel()
    @FocusState private var isFieldFocused: Bool
    @Environment(\.appRouter) private var router

    public init() {}

    public var body: some View {
        Map(position: $viewModel.cameraPosition) { }
            .mapStyle(.imagery(elevation: .realistic))
            .mapControlVisibility(.hidden)
            .ignoresSafeArea()
            .overlay {
                ZStack {
                    VStack {
                        if viewModel.state != .welcome {
                            Header(
                                leftView: {
                                    PinzButton(
                                        type: .icon(.chevronLeft),
                                        tint: .white,
                                        action: .plain { viewModel.dispatch(.back) }
                                    )
                                }
                            )
                        }

                        Spacer()

                        ZStack {
                            switch viewModel.state {
                            case .welcome:
                                welcomeOverlay.transition(.opacity)
                            case .email:
                                emailInputOverlay.transition(.move(edge: .bottom))
                            case .login(let loginState):
                                switch loginState {
                                case .passkeyPrompt:
                                    passkeyPromptOverlay.transition(.move(edge: .bottom))
                                }
                            case .register(let registerState):
                                switch registerState {
                                case .code:
                                    registerCodeOverlay.transition(.move(edge: .bottom))
                                case .nickname:
                                    registerNicknameOverlay.transition(.move(edge: .bottom))
                                case .passkeyPrompt:
                                    passkeyPromptOverlay.transition(.move(edge: .bottom))
                                }
                            }
                        }
                        .if(viewModel.state != .welcome) { view in
                            view
                                .padding(.horizontal, 6)
                                .padding(.bottom, 8)
                                .background {
                                    GradientView(style: .bottom, color: .black, height: 264)
                                }
                        }
                    }
                }
            }
            .animation(.easeInOut(duration: 0.2), value: viewModel.state)
            .onAppear {
                viewModel.setRouter(router)
                viewModel.dispatch(.startRotation)
            }
    }
}

// MARK: - Main flow views

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
                        .frame(width: 78, height: 54)
                        .background(
                            Rectangle()
                                .fill(.white)
                                .cornerRadius(16)
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

    private var passkeyPromptOverlay: some View {
        VStack(spacing: 12) {
            ProgressView()
                .tint(.white)
                .scaleEffect(1.2)
            Text("Подтвердите при помощи Face ID")
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(.white)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 20)
    }
}

// MARK: - Register flow views

extension AuthFlowView {
    private var registerCodeOverlay: some View {
        PinzTextField(
            label: "verification code:",
            style: .segmented(4),
            text: $viewModel.text,
            keyboardType: .numberPad,
            action: .async {
                try await viewModel.asyncDispatch(.continue)
            }
        )
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
