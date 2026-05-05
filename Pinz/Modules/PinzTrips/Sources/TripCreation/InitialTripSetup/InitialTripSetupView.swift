import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase

public struct InitialTripSetupView: View {

    @State private var viewModel: InitialTripSetupViewModel

    @State private var isSeasonPickerPresented = false
    @State private var isCategoryPickerPresented = false

    @State private var isMediaPickerPresented = false
    @State private var pickerItems: [PhotosPickerItem] = []

    let galleryColumns: Int = 3
    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init() {
        viewModel = InitialTripSetupViewModel()
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: viewModel.state == .gallery) {
                header
            } stickyHeader: {
                if !viewModel.isLoading {
                    SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
                        .padding(.horizontal, 12)
                }
            } content: {
                if !viewModel.isLoading {
                    content
                }
            }

            if viewModel.isLoading {
                LoadingView(status: viewModel.loadingStatus?.localizedValue)
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
        }
        .itemsPickerSheet(
            isPresented: $isSeasonPickerPresented,
            items: TripSeason.allCases,
            selection: $viewModel.season,
        )
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: TripCategory.allCases,
            selection: $viewModel.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.category = .custom(value)
            },
        )
        .photosPicker(
            isPresented: $isMediaPickerPresented,
            selection: $pickerItems,
            maxSelectionCount: nil,
            matching: .any(of: [.images, .videos])
        )
        .onChange(of: pickerItems) { _, newItems in
            guard !newItems.isEmpty else { return }
            viewModel.dispatch(.addMedias(newItems))
            pickerItems = []
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            AnimatableHeaderTitle(
                animatableTitle: PinzBaseStrings.TripCreation.Title.main,
                title: $viewModel.name
            )
        }, rightView: {
            if viewModel.state == .gallery {
                PinzButton(
                    type: .icon(.plus),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { isMediaPickerPresented = true }
                )
            }
        })
    }

    private var content: some View {
        Group {
            if viewModel.state == .info {
                ScrollView {
                    VStack(spacing: 12) {
                        nameInput
                        general
                        descriptionInput
                    }.padding(.horizontal, 12)
                }.scrollIndicators(.hidden)
            } else {
                gallery
                    .padding(.horizontal, gallerySpacing)
            }
        }.padding(.bottom, 60)
    }

    @ViewBuilder
    private var nameInput: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "tripNameTextField",
                    text: $viewModel.name,
                    placeholder: PinzBaseStrings.TripCreation.Placeholder.name
                ))
            ],
            subtitle: PinzBaseStrings.TripCreation.Hint.tripNameRules
        )
    }

    private var general: some View {
        SettingsGroup(
            settings: [
                .picker(Setting.PickerSetting(
                    id: "tripSeasonPicker",
                    leading: .iconTitle(TripSeason.icon(for: viewModel.season), viewModel.season.value),
                    isPickerPresented: $isSeasonPickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "tripCategoryPicker",
                    leading: .iconTitle(TripInfoIcon.info, viewModel.category.value),
                    isPickerPresented: $isCategoryPickerPresented
                ))
            ]
        )
    }

    @ViewBuilder
    private var descriptionInput: some View {
        DescriptionEditingView(
            text: Binding(get: {
                viewModel.description ?? ""
            }, set: { value in
                viewModel.description = value
            }),
            placeholder: PinzBaseStrings.TripCreation.Placeholder.description
        )
    }

    var gallery: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: gallerySpacing) {
                PinzGrid($viewModel.medias, columns: galleryColumns, spacing: gallerySpacing) { media, _ in
                    LoadedMediaThumbnailView(
                        media: media,
                        contentMode: .fit,
                        cornerRadius: 14,
                        onMediaDelete: {
                            withAnimation(.easeInOut(duration: 0.3)) {
                                viewModel.dispatch(.deleteMedia(media.id))
                            }
                        }
                    )
                    .onTapGesture {
//                        viewModel.dispatch(.navigate(.mediaInfo(media)))
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.horizontal, gallerySpacing)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.TripCreation.Button.generatePins),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false,
                action: .async {
                    try await viewModel.asyncDispatch(.continue)
                }
            )
//            .disabledWithOpacity(
//                viewModel.name.isEmpty
//                || viewModel.description?.isEmpty == true
//                || viewModel.category == .none
//                || viewModel.season == .none
//                || viewModel.medias.isEmpty
//            )
        }
    }
}
