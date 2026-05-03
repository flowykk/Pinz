import SwiftUI
import PinzUI
import PinzBase

public struct EmailChangeView: View {

    @State private var viewModel: EmailChangeViewModel
    @FocusState private var emailFocused: Bool

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    var codeTextFieldStyle = CodeInputTextField.Style(
        segmentsCount: 4,
        background: PinzUIAsset.backgroundSecondary.color,
        cornerRadius: 16,
        fontSize: 16,
        width: 50,
        height: 64
    )

    public init(
        email: String,
        userId: String? = nil,
        onChangeSuccess: @escaping (String) -> Void
    ) {
        viewModel = EmailChangeViewModel(email: email, userId: userId, successAction: onChangeSuccess)
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

            VStack(spacing: 16) {
                emailInputView
                    .padding(.horizontal, 12)

                if viewModel.state == .code {
                    secondCodeInputView
                }
            }
            Spacer()
            PinzButton(
                type: .slot(style: .primary, title: viewModel.nextButtonTitle),
                disabled: viewModel.isNextButtonDisabled || viewModel.isLoading,
                action: .async { await viewModel.continueTapped() }
            )
            .padding(.horizontal, 12)
            .padding(.bottom, 16)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
            emailFocused = true
        }
    }

    private var emailInputView: some View {
        VStack(spacing: 12) {
            Text(PinzBaseStrings.EmailChange.Label.newEmail)
                .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
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
                .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 64)

            CodeInputTextField(
                code: $viewModel.code,
                style: codeTextFieldStyle,
                onCodeFilled: {}
            )
        }
        .padding(.horizontal, 12)
    }
}
