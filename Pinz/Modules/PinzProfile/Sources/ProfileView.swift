import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

enum ProfileIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case chart = "chart.xyaxis.line"
    case map = "map"
    case heart = "heart"
    case bookmark = "bookmark"

    case bell = "bell.badge"
    case paintbrush = "paintbrush"

    case trash = "trash"
    case door = "door.right.hand.open"
}

public struct ProfileView: View {

    @State private var viewModel: ProfileViewModel

    @State private var imageEditingDialogShown = false
    @State private var photoPickerShown = false
    @State private var isAddPersonPresented = false
    @Environment(\.appRouter) private var router

    public init() {
        viewModel = ProfileViewModel(
            user: User(
                nickname: "flowykk",
                email: "cristgames123@gmail.com"
            )
        )
    }

    public var body: some View {
        VStack(spacing: 0) {
            header

            avatar
                .padding(.top, 12)

            VStack(spacing: 12) {
                switch viewModel.state {
                case .default:
                    defaultSettings
                case .editing:
                    editingSettings
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)

            Spacer()
        }
        .onAppear {
            viewModel.setRouter(router)
//            if let appRouter = router as? AppRouting {
//                appRouter.onEmailUpdate = { [weak viewModel] newEmail in
//                    viewModel?.updateEmail(newEmail)
//                }
//            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .transition(.opacity)
        .confirmationDialog(
            "Выберите действие",
            isPresented: $imageEditingDialogShown,
            titleVisibility: .visible
        ) {
            Button("Выбрать из галереи") {
                photoPickerShown = true
            }
            Button("Удалить фотографию", role: .destructive) { }
        }
        .customImagePicker(show: $photoPickerShown, croppedImage: Binding {
            return viewModel.userImage
        } set: { newImage in
            viewModel.dispatch(.setImage(newImage))
        })
        .fullScreenCover(isPresented: $isAddPersonPresented) {
            AddPersonView()
        }
    }
    
    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(
                leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        viewModel.dispatch(.navigateBack)
                    }
                },
                rightView: {
                    HStack(spacing: 4) {
                        PinzButton(type: .icon(.personAdd), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                            isAddPersonPresented = true
                        }
                        PinzButton(type: .icon(.pencil), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                            viewModel.dispatch(.changeState)
                        }
                    }
                }
            )
        case .editing:
            Header {
                Button {
                    viewModel.dispatch(.changeState)
                } label: {
                    Text("Отмена")
                        .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .padding(.leading, 12)
                }
            } centerView: {
                HeaderTitle("Редактирование профиля")
            } rightView: {
                Button {
                    viewModel.dispatch(.changeState)
                } label: {
                    Text("Готово")
                        .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .padding(.trailing, 12)
                }
            }
        }
    }

    private var avatar: some View {
        VStack {
            Image(uiImage: viewModel.userImage)
                .resizable()
                .scaledToFill()
                .frame(120)
                .cornerRadius(60)
                .clipped()

            Group {
                switch viewModel.state {
                case .default:
                    Text("\(viewModel.user.nickname) • \(viewModel.user.email)")
                case .editing:
                    Button {
                        imageEditingDialogShown = true
                    } label: {
                        Text("Изменить фотографию")
                    }

                }
            }
            .roundedFount(
                size: 16,
                foregroundColor: PinzUIAsset.textSecondary.swiftUIColor
            )
        }
    }

    @ViewBuilder
    private var defaultSettings: some View {
        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Статистика",
                    icon: ProfileIcon.chart,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Путешествия",
                    icon: ProfileIcon.map,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
                .default(.init(
                    title: "Желанные места",
                    icon: ProfileIcon.heart,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
                .default(.init(
                    title: "Сохранённые карты",
                    icon: ProfileIcon.bookmark,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Уведомления",
                    icon: ProfileIcon.bell,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
                .default(.init(
                    title: "Оформление",
                    icon: ProfileIcon.paintbrush,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Удалить аккаунт",
                    icon: ProfileIcon.trash,
                    trailIcon: ProfileIcon.chevronRight,
                    style: .destructive,
                    action: .plain { }
                )),
                .default(.init(
                    title: "Выйти",
                    icon: ProfileIcon.door,
                    trailIcon: ProfileIcon.chevronRight,
                    style: .destructive,
                    action: .plain { }
                )),
            ],
        )
    }

    @ViewBuilder
    private var editingSettings: some View {
        SettingsGroup(
            settings: [
                .textField(.init(
                    id: "nicknameTextField",
                    text: $viewModel.user.nickname,
                    placeholder: "Имя",
                    style: .default
                )),
            ],
            subtitle: "Имя пользователя должно состоять из букв, цифр, точки и подчеркивания"
        ).padding(.bottom, 8)

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Сменить почту",
                    values: [.text(viewModel.user.email)],
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.dispatch(.navigateToEmailChange) }
                )),
            ],
        )
    }
}
