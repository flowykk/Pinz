import SwiftUI
import PinzUI
import PinzNavigation
import PinzDomain

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
        user: User(
            nickname: "flowykk",
            email: "cristgames123@gmail.com"
        )
    )

    @State private var imageEditingDialogShown = false
    @State private var photoPickerShown = false
    @State private var isAddPersonPresented = false

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
                destinationView(for: destination).navigationBarHidden(true)
            }
            .confirmationDialog(
                "Выберите действие",
                isPresented: $imageEditingDialogShown,
                titleVisibility: .visible
            ) {
                Button("Выбрать из галереи") {
                    photoPickerShown = true
                }
                Button("Удалить фотографию", role: .destructive) {
                }
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
    }
    
    @ViewBuilder
    private func destinationView(for destination: ProfileDestination) -> some View {
        switch destination {
        case .statistics:
            StatisticsView()
        case .trips:
            TripsView()
        case .wishlist:
            PlacesWishlistView()
        case .savedMaps:
            SavedMapsView()
        case .notifications:
            NotificationsView()
        case .appearance:
            AppearanceView()
        }
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(
                leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {

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
                .frame(width: 120, height: 120)
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
                    action: .plain { }
                )),
            ],
        )
    }
}
