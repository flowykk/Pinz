import SwiftUI
import PinzUI

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
        VStack(spacing: 0) {
            PinzHeader(
                leftView: {
                    PinzButton(type: .chevronLeft, tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        
                    }
                },
                rightView: {
                    HStack(spacing: 4) {
                        PinzButton(type: .personAdd, tint: PinzUIAsset.textPrimary.swiftUIColor) {

                        }
                        PinzButton(type: .pencil, tint: PinzUIAsset.textPrimary.swiftUIColor) {

                        }
                    }
                }
            )

            ScrollView {
                VStack {
                    Image(uiImage: PinzUIAsset.avatar.image)
                        .resizable()
                        .scaledToFill()
                        .frame(width: 100, height: 100)
                        .cornerRadius(50)
                        .clipped()

                    Text("\(viewModel.email) • \(viewModel.nickname)")
                        .roundedFount(
                            size: 14,
                            foregroundColor: PinzUIAsset.textSecondary.swiftUIColor
                        )
                }.padding(.top, 8)

                VStack {
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
                .padding(.horizontal, 12)
                .padding(.vertical, 12)

                Spacer()
            }
        }
    }
}
