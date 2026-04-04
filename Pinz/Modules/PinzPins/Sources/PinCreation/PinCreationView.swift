import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase

fileprivate enum PinCreationIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case warning = "exclamationmark.triangle.fill"
    case checkmark = "checkmark.circle.fill"

    case info = "info.circle.fill"
    case calendar = "calendar"

    case plus = "plus"
}


public struct PinCreationView: View {

    @State private var viewModel: PinCreationViewModel

    @State var isDescriptionCollapsed = true
    @State var isCategoryPickerPresented = false
    @State var isStartDatePickerPresented = false
    @State var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0
    @State private var isMediaPickerPresented = false
    @State private var pickerItems: [PhotosPickerItem] = []

    let galleryColumns: Int = 3
    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = PinCreationViewModel()
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: viewModel.state == .gallery) {
                header
            } stickyHeader: {
                SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
                    .padding(.horizontal, 12)
            } content: {
                Group {
                    if viewModel.state == .info {
                        ScrollView {
                            VStack(spacing: 12) {
                                general
                                tags
                                descriptionEditing
                            }.padding(.horizontal, 12)
                        }.scrollIndicators(.hidden)
                    } else {
                        gallery
                            .padding(.horizontal, gallerySpacing)
                    }
                }.padding(.bottom, 60)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: PinCategory.allCases,
            selection: $viewModel.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.category = .custom(value)
            }
        )
        .datePickerSheet(
            isPresented: $isStartDatePickerPresented,
            date: $viewModel.startDate,
            pickerHeight: $datePickerHeight
        )
        .datePickerSheet(
            isPresented: $isEndDatePickerPresented,
            date: $viewModel.endDate,
            pickerHeight: $datePickerHeight
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
                animatableTitle: PinzBaseStrings.PinCreation.Title.main,
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

    private var gallery: some View {
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
                        viewModel.dispatch(.navigate(.mediaInfo(media)))
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.horizontal, gallerySpacing)
    }

    @ViewBuilder
    private var general: some View {
        SettingsGroup(
            title: PinzBaseStrings.PinCreation.Header.general,
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "pinNameTextField",
                    text: $viewModel.name,
                    placeholder: PinzBaseStrings.PinCreation.Placeholder.name
                )),
            ],
            subtitle: PinzBaseStrings.PinCreation.Hint.pinNameRules
        )

        SettingsGroup(
            settings: [
                .picker(Setting.PickerSetting(
                    id: "pinCategoryPicker",
                    leading: .iconTitle(PinCreationIcon.info, viewModel.category.value),
                    isPickerPresented: $isCategoryPickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "pinStartDatePicker",
                    leading: .iconTitle(PinCreationIcon.calendar, PinzBaseStrings.Common.Label.startDate),
                    value: .text(viewModel.startDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                    isPickerPresented: $isStartDatePickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "pinEndDatePicker",
                    leading: .iconTitle(PinCreationIcon.calendar, PinzBaseStrings.Common.Label.endDate),
                    value: .text(viewModel.endDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                    isPickerPresented: $isEndDatePickerPresented
                )),
            ]
        )
    }

    private var tags: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle(PinzBaseStrings.PinCreation.Header.tags)
                .padding(.leading, 12)
            TagsView(
                tags: viewModel.tags,
                onTagAdd: { tag in
                    viewModel.dispatch(.addTag(tag))
                },
                onTagDelete: { tag in
                    viewModel.dispatch(.deleteTag(tag))
                },
                style: .editable
            ).padding(.top, 2)
        }
    }

    private var descriptionEditing: some View {
        DescriptionEditingView(
            title: PinzBaseStrings.Common.Label.description,
            text: Binding(get: {
                viewModel.description ?? ""
            }, set: { value in
                viewModel.description = value
            }),
            placeholder: PinzBaseStrings.PinCreation.Placeholder.description
        )
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false,
                action: .async { }
            ).disabledWithOpacity(viewModel.name.isEmpty)
        }
    }
}
