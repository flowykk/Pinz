import SwiftUI
import PinzUI
import PinzDomain

extension PinInfoView {
    var settings: some View {
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
    var general: some View {
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

    var map: some View {
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
}
