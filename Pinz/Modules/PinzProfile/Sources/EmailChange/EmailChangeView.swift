import SwiftUI
import PinzUI

public struct EmailChangeView: View {
    @Environment(\.dismiss) var dismiss

    @State private var viewModel: EmailChangeViewModel
    @FocusState private var emailFocused: Bool

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
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    dismiss()
                }
            }, centerView: {
                HeaderTitle("Смена почты")
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
                                type: .slot(style: .primary, title: "Далее")
                            ) {
                                viewModel.dispatch(.continue)
                            }
                        }
                    }
                }.padding(.horizontal, 12)
            }

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    private var firstCodeInputView: some View {
        VStack(spacing: 16) {
            Text("cristgames123@gmail.com")
                .roundedFount(
                    size: 16,
                    weight: .semibold,
                    foregroundColor: PinzUIAsset.accentRed.swiftUIColor
                )

            Text("Здесь Вы можете сменить почту. Все данные Вашего аккаунта будут сохранены. Для начала введите код, который пришёл на привязанный email")
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 64)

            codeInputView
        }
    }

    private var emailInputView: some View {
        VStack(spacing: 12) {
            Text("Введите новый email")
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)

            SettingsGroup(
                settings: [
                    .textField(Setting.TextFieldSetting(
                        id: "emailTextField",
                        text: $viewModel.email,
                        placeholder: "example@example.com",
                        focused: $emailFocused,
                        style: .default
                    ))
                ]
            )
        }
    }

    private var secondCodeInputView: some View {
        VStack(spacing: 12) {
            Text("Теперь введите код, который пришёл на новый email")
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
