import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain

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
            PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                viewModel.dispatch(.navigate(.back))
            }
        }, centerView: {
            let defaultTitle = "Создание пина"
            HeaderTitle(
                viewModel.name.isEmpty ? defaultTitle : viewModel.name,
                subtitle: viewModel.name.isEmpty ? nil : defaultTitle
            ).animation(.default, value: viewModel.name)
        }, rightView: {
            if viewModel.state == .gallery {
                PinzButton(type: .icon(.plus), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    isMediaPickerPresented = true
                }
            }
        })
    }

    var gallery: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: gallerySpacing) {
                PinzGrid($viewModel.medias, columns: galleryColumns, spacing: gallerySpacing) { media, _ in
                    LoadedMediaThumbnailView(
                        media: media,
                        contentMode: .fit,
                        cornerRadius: 14
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
    var general: some View {
        SettingsGroup(
            title: "Общая информация",
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "pinNameTextField",
                    text: $viewModel.name,
                    placeholder: "Название пина"
                )),
            ],
            subtitle: "Название пина должно состоять из букв, цифр, точки и подчеркивания"
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
                    leading: .iconTitle(PinCreationIcon.calendar, "Дата начала"),
                    value: .text(viewModel.startDate?.formattedToDayMonthYear ?? "Не выбрано"),
                    isPickerPresented: $isStartDatePickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "pinEndDatePicker",
                    leading: .iconTitle(PinCreationIcon.calendar, "Дата конца"),
                    value: .text(viewModel.endDate?.formattedToDayMonthYear ?? "Не выбрано"),
                    isPickerPresented: $isEndDatePickerPresented
                )),
            ]
        )
    }

    private var tags: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle("Теги")
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
            text: Binding(get: {
                viewModel.description ?? ""
            }, set: { value in
                viewModel.description = value
            }),
            placeholder: "Описание пина"
        )
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: "Далее"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false
            ) {
//                viewModel.dispatch(.navigate(.pinCreation))
            }
        }
    }
}
