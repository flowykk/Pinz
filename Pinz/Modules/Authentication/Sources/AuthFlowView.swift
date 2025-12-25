import SwiftUI
import MapKit
import UIComponents

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
                             PinzHeader(
                                 leftView: {
                                     PinzButton(type: .leftChevron, tint: .white) {
                                         DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                                             viewModel.dispatch(.back)
                                         }
                                     }
                                 }
                             )
                         }

                        Spacer()

                        ZStack {
                            switch viewModel.state {
                            case .welcome:
                                welcomeOverlay
                                    .transition(.opacity)
                            case .email:
                                emailInputOverlay
                                    .transition(.move(edge: .bottom))
                            case .auth:
                                authPasswordOverlay
                                    .transition(.move(edge: .bottom))
                            default:
                                welcomeOverlay
                            }
                        }.if(viewModel.state != .welcome) { view in
                            view.background(
                                LinearGradient(
                                    gradient: Gradient(colors: [
                                        Color.black.opacity(0.8),
                                        Color.clear
                                    ]),
                                    startPoint: .bottom,
                                    endPoint: .top
                                ).frame(height: 264)
                            )
                        }
                    }
                }
            }
            .animation(.easeInOut(duration: 0.2), value: viewModel.state)
            .onAppear {
                viewModel.dispatch(.startRotation)
            }
    }
    
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
        VStack {
            PinzTextField(
                label: "email:",
                placeholder: "your@email.com",
                text: $viewModel.email,
                keyboardType: .emailAddress,
                autocapitalization: .never,
                action: .async {
                    try await viewModel.asyncDispatch(.proceedFromEmail)
                }
            )
            .padding(.horizontal, 6)
            .focused($isFieldFocused)
            .padding(.bottom, 8)
        }
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }
    
    private var authPasswordOverlay: some View {
        VStack {
            PinzTextField(
                label: "name:",
                placeholder: "your name",
                text: $viewModel.name,
                keyboardType: .default,
                autocapitalization: .words,
                action: .plain { }
            )
            .padding(.horizontal, 6)
            .focused($isFieldFocused)
            .padding(.bottom, 8)
        }
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                isFieldFocused = true
            }
        }
    }
}

#Preview {
    AuthFlowView()
}

