import SwiftUI
import PhotosUI
import PinzUI

enum WishlistCreationIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case camera = "camera.fill"
}

public struct WishlistCreationView: View {

    @State private var viewModel: WishlistCreationViewModel
    @State private var isPhotoPickerPresented: Bool = false
    @State private var pickerItems: [PhotosPickerItem] = []

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = WishlistCreationViewModel()
    }

    public var body: some View {
        ZStack {
            VStack(spacing: 0) {
                Header(leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        viewModel.dispatch(.navigate(.back))
                    }
                })

                content
                    .padding(.horizontal, 12)
                    .padding(.top, 12)

                Spacer(minLength: 0)
            }

            gradientWithButtons
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

    private var content: some View {
        VStack (spacing: 12) {
            imageUploading
            textFields
        }
    }

    @ViewBuilder
    private var imageUploading: some View {
        if let image = viewModel.image {
            VStack(alignment: .leading, spacing: 4) {
                Button {
                    isPhotoPickerPresented = true
                } label: {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: .fit)
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

    @ViewBuilder
    private var textFields: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "wishlistElementNameTextField",
                    text: $viewModel.name,
                    placeholder: "Название места"
                )),
            ],
            subtitle: "Название места должно состоять из букв, цифр, точки и подчеркивания"
        )

        DescriptionEditingView(
            text: Binding(get: {
                viewModel.description
            }, set: { value in
                viewModel.description = value
            }),
            placeholder: "Описание места"
        )
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: "Готово"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false
            ) {
                viewModel.dispatch(.complete)
            } //.disabledWithOpacity(viewModel.name.isEmpty)
        }
    }
}
