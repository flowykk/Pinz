import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

extension PinInfoView {
    var settings: some View {
        VStack(spacing: 12) {
            if viewModel.pin.isNameCensored {
                Text(PinzBaseStrings.Censorship.Banner.name)
                    .roundedFont(size: 13, foregroundColor: PinzUIAsset.accentRed.swiftUIColor)
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.horizontal, 4)
            }
            if viewModel.isEditing { nameEditing }
            general
            if !viewModel.pin.tags.isEmpty || viewModel.isEditing { tags }
            if viewModel.isEditing {
                descriptionEditing
                delete
            } else {
                description
                privacy
            }
        }
    }

    @ViewBuilder
    var general: some View {
        let startDateValue: String = viewModel.pin.startDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected
        let endDateValue: String = viewModel.pin.endDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected

        let defaultSettings: [Setting] = [
            .default(Setting.DefaultSetting(
                id: "pinCategory",
                leading: .iconTitle(PinInfoIcon.info, PinzBaseStrings.PinInfo.Label.category),
                trailing: .valuesIcon([.text(viewModel.pin.category.value)], PinInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.edit) }
            )),
            .default(Setting.DefaultSetting(
                id: "pinDates",
                leading: .iconTitle(PinInfoIcon.calendar, PinzBaseStrings.PinInfo.Label.dates),
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
                leading: .iconTitle(PinInfoIcon.calendar, PinzBaseStrings.Common.Label.startDate),
                value: .text(startDateValue),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "pinEndDatePicker",
                leading: .iconTitle(PinInfoIcon.calendar, PinzBaseStrings.Common.Label.endDate),
                value: .text(endDateValue),
                isPickerPresented: $isEndDatePickerPresented
            )),
        ]

        SettingsGroup(
            title: PinzBaseStrings.PinCreation.Header.general,
            settings: viewModel.isEditing ? editingSettings : defaultSettings
        )
    }

    private var nameEditing: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "pinNameTextField",
                    text: $viewModel.pin.name,
                    placeholder: PinzBaseStrings.PinCreation.Placeholder.name
                )),
            ],
            subtitle: PinzBaseStrings.PinCreation.Hint.pinNameRules
        )
    }

    private var tags: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle(PinzBaseStrings.PinCreation.Header.tags)
                .padding(.leading, 12)
            TagsView(
                tags: viewModel.pin.tags,
                onTagAdd: { tag in
                    viewModel.dispatch(.addTag(tag))
                },
                onTagDelete: { tag in
                    viewModel.dispatch(.deleteTag(tag))
                },
                style: viewModel.isEditing ? .editable : .default,
                maxTags: PinInfoViewModel.pinTagsMaxCount
            ).padding(.top, 2)
        }
    }

    private var description: some View {
        DescriptionView(
            description: viewModel.pin.description,
            subtitle: viewModel.pin.isDescriptionCensored ? PinzBaseStrings.Censorship.Banner.description : nil,
            subtitleStyle: viewModel.pin.isDescriptionCensored ? .destructive : .default,
            onAddAction: {
                viewModel.dispatch(.edit)
            }
        )
    }

    private var descriptionEditing: some View {
        DescriptionEditingView(
            title: PinzBaseStrings.Common.Label.description,
            subtitle: viewModel.pin.isDescriptionCensored
                ? PinzBaseStrings.Censorship.Banner.description
                : nil,
            subtitleStyle: viewModel.pin.isDescriptionCensored ? .destructive : .default,
            text: Binding(get: {
                viewModel.pin.description ?? ""
            }, set: { value in
                viewModel.pin.description = value
            }),
            placeholder: PinzBaseStrings.PinCreation.Placeholder.description,
            textFieldId: "pinDescriptionEditingTextField"
        )
    }

    private var privacy: some View {
        PrivacySection(
            initialSelection: PrivacyIcon.from(isPrivate: viewModel.pin.isPrivate),
            onSelectionChanged: { [viewModel] in viewModel.dispatch(.updatePrivacy($0)) }
        )
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
                leading: .iconTitle(PinInfoIcon.trash, PinzBaseStrings.PinInfo.Button.delete),
                trailing: .icon(PinInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { showDeletePinAlert = true }
            ))
        ])
    }
}
