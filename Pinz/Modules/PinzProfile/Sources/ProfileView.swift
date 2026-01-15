import SwiftUI
import PinzUI
import PinzNavigation

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

    @State private var viewModel = ProfileViewModel(
        nickname: "flowykk",
        email: "cristgames123@gmail.com"
    )

    public init() {}
    
    public var body: some View {
        NavigationStack(path: $viewModel.navigator.path) {
            VStack(spacing: 0) {
                header

                ScrollView {
                    avatar
                        .padding(.top, 12)

                    VStack {
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
            }
            .transition(.opacity)
            .navigationDestination(for: ProfileDestination.self) { destination in
                destinationView(for: destination)
            }
            .navigationBarHidden(true)
        }
    }
    
    @ViewBuilder
    private func destinationView(for destination: ProfileDestination) -> some View {
        switch destination {
        case .addPerson:
            AddPersonView()
        case .statistics:
            Text("Statistics")
        case .trips:
            Text("Trips")
        case .wishlist:
            Text("Wishlist")
        case .savedMaps:
            Text("Saved Maps")
        case .notifications:
            Text("Notifications")
        case .appearance:
            Text("Appearance")
        }
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            PinzHeader(
                leftView: {
                    PinzButton(type: .chevronLeft, tint: PinzUIAsset.textPrimary.swiftUIColor) {

                    }
                },
                rightView: {
                    HStack(spacing: 4) {
                        PinzButton(type: .personAdd, tint: PinzUIAsset.textPrimary.swiftUIColor) {
                            viewModel.navigator.navigate(to: .addPerson)
                        }
                        PinzButton(type: .pencil, tint: PinzUIAsset.textPrimary.swiftUIColor) {
                            viewModel.dispatch(.changeState)
                        }
                    }
                }
            )
        case .editing:
            PinzHeader(
                leftView: {
                    Button {
                        viewModel.dispatch(.changeState)
                    } label: {
                        Text("Отмена")
                            .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                            .padding(.leading, 12)
                    }
                },
                centerView: {
                    Text("Редактирование профиля")
                        .roundedFount(size: 18, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                },
                rightView: {
                    Button {
                        viewModel.dispatch(.changeState)
                    } label: {
                        Text("Готово")
                            .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                            .padding(.trailing, 12)
                    }
                }
            )
        }
    }

    private var avatar: some View {
        VStack {
            Image(uiImage: PinzUIAsset.avatar.image)
                .resizable()
                .scaledToFill()
                .frame(width: 120, height: 120)
                .cornerRadius(60)
                .clipped()

            Group {
                switch viewModel.state {
                case .default:
                    Text("\(viewModel.nickname) • \(viewModel.email)")
                case .editing:
                    Text("Изменить фотографию")
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
                    action: .plain { viewModel.navigator.navigate(to: .statistics) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Путешествия",
                    icon: ProfileIcon.map,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.navigator.navigate(to: .trips) }
                )),
                .default(.init(
                    title: "Желанные места",
                    icon: ProfileIcon.heart,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.navigator.navigate(to: .wishlist) }
                )),
                .default(.init(
                    title: "Сохранённые карты",
                    icon: ProfileIcon.bookmark,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.navigator.navigate(to: .savedMaps) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(.init(
                    title: "Уведомления",
                    icon: ProfileIcon.bell,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.navigator.navigate(to: .notifications) }
                )),
                .default(.init(
                    title: "Оформление",
                    icon: ProfileIcon.paintbrush,
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { viewModel.navigator.navigate(to: .appearance) }
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
                    text: $viewModel.nickname,
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
                    values: [.text(viewModel.email)],
                    trailIcon: ProfileIcon.chevronRight,
                    action: .plain { }
                )),
            ],
        )
    }
}
