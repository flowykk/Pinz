import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase

enum WishlistElementIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case trash = "trash"
}


public struct WishlistElementView: View {

    @State private var viewModel: WishlistElementViewModel
    @State private var isPhotoPickerPresented: Bool = false
    @State private var pickerItems: [PhotosPickerItem] = []

    @Environment(\.appRouter) private var router

    public init(element: WishlistElement) {
        viewModel = WishlistElementViewModel(element: element)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            settings
                .padding(.horizontal, 12)
                .padding(.bottom, 60)
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
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(viewModel.element.title)
            }, rightView: {
                PinzButton(
                    type: .icon(.pencil),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.edit) }
                )
            })
        case .editing:
            Header {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.cancel),
                    action: .plain { viewModel.dispatch(.endEdit) }
                )
            } rightView: {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.done),
                    action: .plain { viewModel.dispatch(.endEdit) }
                )
            }
        }
    }

    private var settings: some View {
        VStack(spacing: 12) {
            image
            if viewModel.state == .default {
                description
            } else {
                nameEditing
                descriptionEditing
                delete
            }
        }
    }

    private var image: some View {
        VStack(alignment: .leading, spacing: 4) {
            Button {
                if viewModel.state == .default {
                    viewModel.dispatch(.edit)
                } else {
                    isPhotoPickerPresented = true
                }
            } label: {
                CollapsibleImageView(image: viewModel.element.image)
            }
            .buttonStyle(.plain)

            if viewModel.state == .editing {
                SettingSubtitle(PinzBaseStrings.WishlistElement.Hint.changePhoto)
                    .padding(.leading, 12)
            }
        }
    }

    private var nameEditing: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "wishlistElementNameTextField",
                    text: $viewModel.element.title,
                    placeholder: PinzBaseStrings.WishlistElement.Label.title
                )),
            ],
            subtitle: PinzBaseStrings.WishlistElement.Hint.titleRules
        )
    }

    private var description: some View {
        DescriptionView(
            description: viewModel.element.description,
            onAddAction: {
                viewModel.dispatch(.edit)
            }
        )
    }

    private var descriptionEditing: some View {
        DescriptionEditingView(
            title: PinzBaseStrings.WishlistElement.Label.description,
            text: Binding(get: {
                viewModel.element.description
            }, set: { value in
                viewModel.element.description = value
            }),
            placeholder: PinzBaseStrings.WishlistElement.Placeholder.description
        )
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "wishlistElementDelete",
                leading: .iconTitle(WishlistElementIcon.trash, PinzBaseStrings.WishlistElement.Button.delete),
                trailing: .icon(WishlistElementIcon.chevronRight),
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
