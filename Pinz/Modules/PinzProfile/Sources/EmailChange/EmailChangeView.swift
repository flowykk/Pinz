import SwiftUI
import PinzUI
import PinzBase

public struct EmailChangeView: View {

    @State private var viewModel: EmailChangeViewModel
    @FocusState private var emailFocused: Bool

    @Environment(\.appRouter) private var router

    var codeTextFieldStyle = CodeInputTextField.Style(
        segmentsCount: 4,
        background: PinzUIAsset.backgroundSecondary.color,
        cornerRadius: 16,
        fontSize: 16,
        width: 50,
        height: 64
    )

    public init(email: String, onChangeSuccess: @escaping (String) -> Void) {
        viewModel = EmailChangeViewModel(email: email, successAction: onChangeSuccess)
    }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(PinzBaseStrings.EmailChange.Title.main)
            })

            Spacer()

            switch viewModel.state {
            case .firstCode:
                firstCodeInputView
            default:
                ZStack {
                    VStack(spacing: 28) {
                        emailInputView

                        if viewModel.state == .secondCode {
                            secondCodeInputView
                        }
                    }

                    if viewModel.state == .email {
                        VStack {
                            Spacer()
                            PinzButton(
                                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                                action: .plain { viewModel.dispatch(.continue) }
                            )
                        }
                    }
                }.padding(.horizontal, 12)
            }

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
        }
    }

    private var firstCodeInputView: some View {
        VStack(spacing: 16) {
            Text("cristgames123@gmail.com")
                .roundedFount(
                    size: 16,
                    weight: .semibold,
                    foregroundColor: PinzUIAsset.accentRed.swiftUIColor
                )

            Text(PinzBaseStrings.EmailChange.Description.instructions)
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 64)

            codeInputView
        }
    }

    private var emailInputView: some View {
        VStack(spacing: 12) {
            Text(PinzBaseStrings.EmailChange.Label.newEmail)
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)

            SettingsGroup(
                settings: [
                    .textField(Setting.TextFieldSetting(
                        id: "emailTextField",
                        text: $viewModel.email,
                        placeholder: PinzBaseStrings.EmailChange.Placeholder.email,
                        focused: $emailFocused,
                        style: .default
                    ))
                ]
            )
        }
    }

    private var secondCodeInputView: some View {
        VStack(spacing: 12) {
            Text(PinzBaseStrings.EmailChange.Description.verificationCode)
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 64)

            codeInputView
        }
    }

    private var codeInputView: some View {
        CodeInputTextField(
            code: $viewModel.code,
            style: codeTextFieldStyle,
            onCodeFilled: {
                emailFocused = true
                viewModel.dispatch(.continue)
            }
        )
    }
}
