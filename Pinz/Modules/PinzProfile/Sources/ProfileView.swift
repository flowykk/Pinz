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
    case opticaldiscdrive = "opticaldiscdrive"

    case trash = "trash"
    case door = "door.right.hand.open"
}

public struct ProfileView: View {

    @State var viewModel: ProfileViewModel
    
    @State var cacheSize = ""
    @State var showClearCacheAlert = false

    @State var imageEditingDialogShown = false
    @State var photoPickerShown = false
    @State var isAddPersonPresented = false
    @Environment(\.appRouter) private var router

    var accountDeleteSetting: Setting {
        .default(Setting.DefaultSetting(
            id: "profileDelete",
            leading: .iconTitle(ProfileIcon.trash, PinzBaseStrings.Profile.Button.deleteAccount),
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
        .onAppear { 
            viewModel.setRouter(router)
            cacheSize = FileManagerImageStorage.shared.getCacheSize()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .transition(.opacity)
        .confirmationDialog(
            PinzBaseStrings.Common.Alert.Title.selectAction,
            isPresented: $imageEditingDialogShown,
            titleVisibility: .visible
        ) {
            Button(PinzBaseStrings.Common.Button.selectFromGallery) {
                photoPickerShown = true
            }
            Button(PinzBaseStrings.Common.Button.deletePhoto, role: .destructive) { }
        }
        .customImagePicker(show: $photoPickerShown, croppedImage: Binding {
            return viewModel.userImage
        } set: { newImage in
            viewModel.dispatch(.setImage(newImage))
        })
        .fullScreenCover(isPresented: $isAddPersonPresented) {
            AddPersonView()
        }
        .alert(PinzBaseStrings.Alert.ClearCache.title, isPresented: $showClearCacheAlert) {
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) { }
            Button(PinzBaseStrings.Alert.ClearCache.confirm, role: .destructive) {
                FileManagerImageStorage.shared.clear()
                cacheSize = FileManagerImageStorage.shared.getCacheSize()
            }
        } message: {
            Text(PinzBaseStrings.Alert.ClearCache.message)
        }
    }
    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(
                leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                },
                rightView: {
                    HStack(spacing: 4) {
                        PinzButton(
                            type: .icon(.personAdd),
                            tint: PinzUIAsset.textPrimary.swiftUIColor,
                            action: .plain { isAddPersonPresented = true }
                        )
                        PinzButton(
                            type: .icon(.pencil),
                            tint: PinzUIAsset.textPrimary.swiftUIColor,
                            action: .plain { viewModel.dispatch(.changeState) }
                        )
                    }
                }
            )
        case .editing:
            Header {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.cancel),
                    action: .plain { viewModel.dispatch(.changeState) }
                )
            } centerView: {
                HeaderTitle(PinzBaseStrings.Profile.Title.editProfile)
            } rightView: {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.done),
                    action: .plain { viewModel.dispatch(.changeState) }
                )
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
                        Text(PinzBaseStrings.Common.Button.editPhoto)
                    }
                }
            }
            .roundedFont(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
        }
    }

    @ViewBuilder
    private var defaultSettings: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileStats",
                    leading: .iconTitle(ProfileIcon.chart, PinzBaseStrings.Profile.Label.statistics),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.statistics)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileTrips",
                    leading: .iconTitle(ProfileIcon.map, PinzBaseStrings.Profile.Label.trips),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.trips)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileWishlist",
                    leading: .iconTitle(ProfileIcon.heart, PinzBaseStrings.Profile.Label.wishlist),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.wishlist)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileSavedMaps",
                    leading: .iconTitle(ProfileIcon.bookmark, PinzBaseStrings.Profile.Label.savedMaps),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.saved)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileNotifications",
                    leading: .iconTitle(ProfileIcon.bell, PinzBaseStrings.Profile.Label.notifications),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.notifications)) }
                )),
                .default(Setting.DefaultSetting(
                    id: "profileAppearance",
                    leading: .iconTitle(ProfileIcon.paintbrush, PinzBaseStrings.Profile.Label.appearance),
                    trailing: .icon(ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.appearance)) }
                )),
            ],
        )

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileClearCache",
                    leading: .iconTitle(ProfileIcon.opticaldiscdrive, PinzBaseStrings.Profile.Label.clearCache),
                    trailing: .valuesIcon([.text(cacheSize)], ProfileIcon.chevronRight),
                    action: .plain { showClearCacheAlert = true }
                )),
            ]
        )

        SettingsGroup(
            settings: [
                accountDeleteSetting,
                .default(Setting.DefaultSetting(
                    id: "profileLeave",
                    leading: .iconTitle(ProfileIcon.door, PinzBaseStrings.Profile.Button.logout),
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
                    placeholder: PinzBaseStrings.Profile.Placeholder.name,
                    style: .default
                )),
            ],
            subtitle: PinzBaseStrings.Profile.Hint.nicknameRules
        ).padding(.bottom, 8)

        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "profileEmailChanging",
                    leading: .title(PinzBaseStrings.Profile.Label.changeEmail),
                    trailing: .valuesIcon([.text(viewModel.user.email)], ProfileIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.emailChange)) }
                )),
            ],
        )

        SettingsGroup(settings: [accountDeleteSetting])
    }
}
