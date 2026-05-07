import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase
import PinzAccessibility

enum WishlistCreationIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case camera = "camera.fill"
}

public struct WishlistElementCreationView: View {

    @State private var viewModel: WishlistElementCreationViewModel
    @State private var isPhotoPickerPresented: Bool = false
    @State private var pickerItems: [PhotosPickerItem] = []

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init(onCreated: @escaping (DesiredPlace) -> Void) {
        viewModel = WishlistElementCreationViewModel(onCreated: onCreated)
    }

    public var body: some View {
        ZStack {
            VStack {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                    .pinzA11y(.wishlist(.button(.back)))
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
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
        }
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
                    placeholder: PinzBaseStrings.Wishlist.Placeholder.placeName
                ))
            ],
            subtitle: PinzBaseStrings.Wishlist.Hint.placeNameRules
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
            placeholder: PinzBaseStrings.Wishlist.Placeholder.placeDescription
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

                SettingSubtitle(PinzBaseStrings.Wishlist.Hint.changePhoto)
                    .padding(.leading, 12)
            }
        } else {
            SettingsGroup(
                settings: [
                    .default(Setting.DefaultSetting(
                        id: "wishlistElementPhoto",
                        leading: .iconTitle(WishlistCreationIcon.camera, PinzBaseStrings.Wishlist.Label.uploadPhoto),
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
                type: .slot(style: .primary, title: viewModel.state == .photo ? PinzBaseStrings.Common.Button.done : PinzBaseStrings.Common.Button.next),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false,
                action: .plain { viewModel.dispatch(.continue) }
            )
            .pinzA11y(.wishlist(.button(.done)))
            .disabledWithOpacity(viewModel.isCompleteButtonDisabled || viewModel.isLoading)
            .animation(.easeInOut(duration: 0.3), value: viewModel.isCompleteButtonDisabled || viewModel.isLoading)
        }
    }
}
