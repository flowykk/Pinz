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

    @State var viewModel: ProfileViewModel

    @State var imageEditingDialogShown = false
    @State var photoPickerShown = false
    @State var isAddPersonPresented = false
    @Environment(\.appRouter) private var router

    var accountDeleteSetting: Setting {
        .default(Setting.DefaultSetting(
            id: "profileDelete",
            leading: .iconTitle(ProfileIcon.trash, "Удалить аккаунт"),
            trailing: .icon(ProfileIcon.chevronRight),
            style: .destructive,
            action: .plain { }
        ))
    }

    public init(user: User) {
        viewModel = ProfileViewModel(user: user)
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
        .onAppear { viewModel.setRouter(router) }
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
                        viewModel.dispatch(.navigate(.back))
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
                PinzButton(type: .text("Отмена")) {
                    viewModel.dispatch(.changeState)
                }
            } centerView: {
                HeaderTitle("Редактирование профиля")
            } rightView: {
                PinzButton(type: .text("Готово")) {
                    viewModel.dispatch(.changeState)
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
            .roundedFount(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
        }
    }

    @ViewBuilder
    private var defaultSettings: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileStats",
                    leading: .iconTitle(ProfileIcon.chart, "Статистика"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.statistics)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileTrips",
                    leading: .iconTitle(ProfileIcon.map, "Путешествия"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.trips)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileWishlist",
                    leading: .iconTitle(ProfileIcon.heart, "Желанные места"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.wishlist)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileSavedMaps",
                    leading: .iconTitle(ProfileIcon.bookmark, "Сохранённые карты"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.saved)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileNotifications",
                    leading: .iconTitle(ProfileIcon.bell, "Уведомления"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.notifications)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileAppearance",
                    leading: .iconTitle(ProfileIcon.paintbrush, "Оформление"),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.appearance)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                accountDeleteSetting,
                .default(Setting.DefaultSetting(
                    id: "profileLeave",
                    leading: .iconTitle(ProfileIcon.door, "Выйти"),
                    trailing: .icon(ProfileIcon.chevronRight),
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
                .textField(Setting.TextFieldSetting(
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
                .default(Setting.DefaultSetting(
                    id: "profileEmailChanging",
                    leading: .title("Сменить почту"),
                    trailing: .valuesIcon([.text(viewModel.user.email)], ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.emailChange)) }
                )),
            ],
        )

        SettingsGroup(settings: [accountDeleteSetting])
    }
}
