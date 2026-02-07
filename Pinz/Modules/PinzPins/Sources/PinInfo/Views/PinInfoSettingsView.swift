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
                leading: .iconTitle(PinInfoIcon.info, "Категория"),
                trailing: .valuesIcon([.text(viewModel.pin.category.value)], PinInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.edit) }
            )),
            .default(Setting.DefaultSetting(
                id: "pinDates",
                leading: .iconTitle(PinInfoIcon.calendar, "Даты"),
                trailing: .valuesIcon([.text(datesSettingValue)], PinInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.edit) }
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "pinCategoryPicker",
                leading: .iconTitle(PinInfoIcon.info, viewModel.pin.category.value),
                isPickerPresented: $isCategoryPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "pinStartDatePicker",
                leading: .iconTitle(PinInfoIcon.calendar, "Дата начала"),
                value: .text(viewModel.pin.startDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "pinEndDatePicker",
                leading: .iconTitle(PinInfoIcon.calendar, "Дата конца"),
                value: .text(viewModel.pin.endDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isEndDatePickerPresented
            )),
        ]

        SettingsGroup(
            title: "Общая информация",
            settings: viewModel.isEditing ? editingSettings : defaultSettings
        )
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
            TagsView(
                tags: viewModel.pin.tags,
                onTagAdd: { tag in
                    viewModel.dispatch(.addTag(tag))
                },
                onTagDelete: { tag in
                    viewModel.dispatch(.deleteTag(tag))
                },
                style: viewModel.isEditing ? .editing : .default
            ).padding(.top, 2)
        }
    }

    private var privacy: some View {
        PrivacySection(members: TripMember.stubs())
    }

    var map: some View {
        Button {
            viewModel.dispatch(.navigate(.changePlace))
        } label: {
            PinPlaceSectionView(pin: $viewModel.pin)
        }
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "pinDelete",
                leading: .iconTitle(PinInfoIcon.trash, "Удалить пин"),
                trailing: .icon(PinInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
