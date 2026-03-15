import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain

enum WishlistCreationIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case camera = "camera.fill"
}

public struct WishlistCreationView: View {

    @State private var viewModel: WishlistCreationViewModel
    @State private var isPhotoPickerPresented: Bool = false
    @State private var pickerItems: [PhotosPickerItem] = []

    @Environment(\.appRouter) private var router

    public init(onCreated: @escaping (WishlistElement) -> Void) {
        viewModel = WishlistCreationViewModel(onCreated: onCreated)
    }

    public var body: some View {
        ZStack {
            VStack {
                Header(leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        viewModel.dispatch(.navigate(.back))
                    }
                })
                Spacer()
                gradientWithButtons
            }

            VStack {
                if viewModel.state == .photo {
                    photoUploading
                } else {
                    VStack(spacing: 12) {
                        nameInput
                        if viewModel.state == .description {
                            descriptionInput
                        }
                    }
                }
            }.padding(.horizontal, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .photosPicker(
            isPresented: $isPhotoPickerPresented,
            selection: $pickerItems,
            maxSelectionCount: 1,
            matching: .images
        )
        .onChange(of: pickerItems) { _, newItems in
            guard let item = newItems.first else { return }
            viewModel.dispatch(.selectPhoto(item))
            pickerItems = []
        }
    }

    @ViewBuilder
    private var nameInput: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "wishlistElementNameTextField",
                    text: $viewModel.name,
                    placeholder: "Название места"
                ))
            ],
            subtitle: "Название места должно состоять из букв, цифр, точки и подчеркивания"
        )
    }

    @ViewBuilder
    private var descriptionInput: some View {
        DescriptionEditingView(
            text: Binding(get: {
                viewModel.description
            }, set: { value in
                viewModel.description = value
            }),
            placeholder: "Описание места"
        )
    }

    @ViewBuilder
    private var photoUploading: some View {
        if let image = viewModel.image {
            VStack(alignment: .center, spacing: 4) {
                Button {
                    isPhotoPickerPresented = true
                } label: {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(maxWidth: .infinity)
                        .cornerRadius(24)
                }.buttonStyle(.plain)

                SettingSubtitle("Нажмите на фото, чтобы поменять его")
                    .padding(.leading, 12)
            }
        } else {
            SettingsGroup(
                settings: [
                    .default(Setting.DefaultSetting(
                        id: "wishlistElementPhoto",
                        leading: .iconTitle(WishlistCreationIcon.camera, "Загрузите фотографию места"),
                        trailing: .icon(WishlistCreationIcon.chevronRight),
                        action: .plain { isPhotoPickerPresented = true }
                    ))
                ]
            )
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: viewModel.state == .photo ? "Готово" : "Далее"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false
            ) {
                viewModel.dispatch(.continue)
            }
            .disabledWithOpacity(viewModel.isCompleteButtonDisabled)
            .animation(.easeInOut(duration: 0.3), value: viewModel.isCompleteButtonDisabled)
        }
    }
}
