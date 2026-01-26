import SwiftUI
import PinzUI
import PinzDomain

enum PinInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case warning = "exclamationmark.triangle.fill"
    case checkmark = "checkmark.circle.fill"

    case info = "info.circle.fill"
    case calendar = "calendar"

    case trash = "trash"
}

public struct PinInfoView: View {

    @State private var viewModel: PinInfoViewModel
    @State private var isDescriptionCollapsed = true

    @State private var isCategoryPickerPresented = false
    @State private var isStartDatePickerPresented = false
    @State private var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0

    @Environment(\.appRouter) private var router

    var datesSettingValue: String {
        if let startDate = viewModel.pin.startDate, let endDate = viewModel.pin.endDate {
            "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            "Не выбрано"
        }
    }

    public init(pin: Pin) {
        viewModel = PinInfoViewModel(pin: pin)
    }

    public var body: some View {
        VStack(spacing: 0) {
            header

            ScrollView {
                VStack(spacing: 0) {
                    switch viewModel.state {
                    case .info, .editing:
                        settings
                    case .gallery:
                        EmptyView()
                    }
                    map.if(viewModel.state != .info) { view in view.hidden() }
                        .padding(.top, 12)
                }
                .padding(.top, 8)
                .padding(.horizontal, 12)

                Spacer()
            }
            .scrollIndicators(.hidden)
            .scrollDisabled(viewModel.isEditing)
        }
        .onAppear { viewModel.setRouter(router) }
        .background(PinzUIAsset.background.swiftUIColor)
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: PinCategory.allCases,
            selection: $viewModel.pin.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.pin.category = .custom(value)
            }
        )
        .datePickerSheet(
            isPresented: $isStartDatePickerPresented,
            date: $viewModel.pin.startDate,
            pickerHeight: $datePickerHeight
        )
        .datePickerSheet(
            isPresented: $isEndDatePickerPresented,
            date: $viewModel.pin.endDate,
            pickerHeight: $datePickerHeight
        )
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .info, .gallery:
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.back)
                }
            }, centerView: {
                HeaderTitle(viewModel.pin.name, subtitle: viewModel.pin.category.value)
            }, rightView: {
                PinzButton(type: .icon(.warning), tint: PinzUIAsset.accentOrange.swiftUIColor) {

                }
                PinzButton(type: .icon(.pencil), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.changeState(.editing))
                }
            }, additionalView: {
                SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
            }, height: nil)
        case .editing:
            Header {
                PinzButton(type: .text("Отмена")) {
                    viewModel.dispatch(.changeState(.info))
                }
            } rightView: {
                PinzButton(type: .text("Готово")) {
                    viewModel.dispatch(.changeState(.info))
                }
            }
        }
    }

    private var settings: some View {
        VStack(spacing: 12) {
            if viewModel.isEditing { nameEditing }
            general
            tags
            if viewModel.isEditing {
                delete
            } else {
                privacy
            }
        }
    }

    @ViewBuilder
    private var general: some View {
        let defaultSettings: [Setting] = [
            .default(Setting.DefaultSetting(
                id: "pinCategory",
                title: "Категория",
                icon: PinInfoIcon.info,
                values: [.text(viewModel.pin.category.value)],
                trailIcon: PinInfoIcon.chevronRight,
                action: .plain { viewModel.dispatch(.changeState(.editing)) }
            )),
            .default(Setting.DefaultSetting(
                id: "pinDates",
                title: "Даты",
                icon: PinInfoIcon.calendar,
                values: [.text(datesSettingValue)],
                trailIcon: PinInfoIcon.chevronRight,
                action: .plain { viewModel.dispatch(.changeState(.editing)) }
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "pinCategoryPicker",
                title: viewModel.pin.category.value,
                icon: PinInfoIcon.info,
                isPickerPresented: $isCategoryPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "pinStartDatePicker",
                title: "Дата начала",
                icon: PinInfoIcon.calendar,
                value: .text(viewModel.pin.startDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "pinEndDatePicker",
                title: "Дата конца",
                icon: PinInfoIcon.calendar,
                value: .text(viewModel.pin.endDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isEndDatePickerPresented
            )),
        ]

        SettingsGroup(
            title: "Общая информация",
            settings: viewModel.isEditing ? editingSettings : defaultSettings
        )//.animation(.default, value: viewModel.pin)
    }

    private var nameEditing: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "pinNameTextField",
                    text: $viewModel.pin.name,
                    placeholder: "Название пина"
                )),
            ],
            subtitle: "Название пина должно состоять из букв, цифр, точки и подчеркивания"
        )
    }

    private var tags: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle("Теги")
                .padding(.leading, 12)
            TagsView(tags: viewModel.pin.tags, onTagAdd: {_ in }, onTagDelete: {_ in })
                .padding(.top, 2)
        }
    }

    private var privacy: some View {
        PrivacySection(
            members: [
                TripMember(isPrivate: true, username: "flowykk"),
                TripMember(isPrivate: false, username: "kostik"),
            ]
        )
    }

    private var map: some View {
        PinPlaceSectionView()
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "pinDelete",
                title: "Удалить пин",
                icon: PinInfoIcon.trash,
                trailIcon: PinInfoIcon.chevronRight,
                style: .destructive,
                action: .plain { }
            ))
        ])
    }

     /*

    private var description: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 0) {
                SettingTitle("Описание")
                if viewModel.trip.description != "" {
                    Spacer()
                    Button {
                        withAnimation(.easeInOut(duration: 0.3)) {
                            isDescriptionCollapsed.toggle()
                        }
                    } label: {
                        HStack(spacing: 4) {
                            Text(isDescriptionCollapsed ? "Раскрыть" : "Скрыть")
                            Image(systemName: "chevron.down")
                                .rotationEffect(.degrees(isDescriptionCollapsed ? 0 : 180))
                        }.roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    }
                }
            }
            .padding(.bottom, 6)
            .padding(.leading, 12)
            .padding(.trailing, 16)

            if !viewModel.trip.description.isEmpty {
                VStack(spacing: 0) {
                    Text(viewModel.trip.description)
                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .lineLimit(isDescriptionCollapsed ? 5 : nil)
                        .frame(maxWidth: .infinity)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                }
                .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
                .cornerRadius(26)
            } else {
                SettingsGroup(settings: [
                    .default(Setting.DefaultSetting(
                        id: "tripDescription",
                        title: "Добавить описание",
                        icon: TripInfoIcon.text,
                        trailIcon: TripInfoIcon.chevronRight,
                        action: .plain {
                            // switch to editing
                        }
                    ))
                ])
            }
        }
    }

    private var pins: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "tripPins",
                    title: "Пины путешествия",
                    icon: TripInfoIcon.pins,
                    trailIcon: TripInfoIcon.chevronRight,
                    action: .plain {  }
                )),
            ],
        )
    }

    @ViewBuilder
    private var general: some View {
        let defaultSettings: [Setting] = [
            .default(Setting.DefaultSetting(
                id: "tripSeason",
                title: "Сезон",
                icon: tripSeasonIcon,
                values: [.text(viewModel.trip.season.value)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripCategory",
                title: "Категория",
                icon: TripInfoIcon.info,
                values: [.text(viewModel.trip.category.value)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripDates",
                title: "Даты",
                icon: TripInfoIcon.calendar,
                values: [.text(datesSettingValue)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain { viewModel.dispatch(.changeState) }
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "tripSeasonPicker",
                title: viewModel.trip.season.value,
                icon: tripSeasonIcon,
                isPickerPresented: $isSeasonPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripCategoryPicker",
                title: viewModel.trip.category.value,
                icon: TripInfoIcon.info,
                isPickerPresented: $isCategoryPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripStartDatePicker",
                title: "Дата начала",
                icon: TripInfoIcon.calendar,
                value: .text(viewModel.trip.startDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripEndDatePicker",
                title: "Дата конца",
                icon: TripInfoIcon.calendar,
                value: .text(viewModel.trip.endDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isEndDatePickerPresented
            )),
        ]

        SettingsGroup(
            title: "Общая информация",
            settings: viewModel.state == .default ? defaultSettings : editingSettings
        ).animation(.default, value: viewModel.trip)
    }

    private var publishing: some View {
        SettingsGroup(
            title: "Публичная информация",
            settings: [
                .default(Setting.DefaultSetting(
                    id: "tripPublishing",
                    title: "Опубликовать путешествие",
                    icon: TripInfoIcon.paperplane,
                    trailIcon: TripInfoIcon.chevronRight,
                    action: .plain {}
                )),
            ],
            subtitle: "Когда нельзя публиковать, можно в сабтайтле это писать"
        )
    }

    private var nameEditing: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "nicknameTextField",
                    text: $viewModel.trip.name,
                    placeholder: "Имя",
                    style: .default
                )),
            ],
            subtitle: "Название путешествия должно состоять из букв, цифр, точки и подчеркивания"
        )
    }

    private var descriptionEditing: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle("Описание")
                .padding(.bottom, 6)
                .padding(.leading, 12)

            SettingsGroup(settings: [
                .textField(Setting.TextFieldSetting(
                    id: "tripDescriptionEditingTextField",
                    text: $viewModel.trip.description,
                    placeholder: "Описание путешествия",
                    style: .multiline
                ))
            ])
        }
    }
    */
}
