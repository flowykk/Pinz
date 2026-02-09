import SwiftUI
import PinzUI
import PinzDomain

enum TripInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case pins = "pin.fill"

    case text = "text.alignleft"

    case sun = "sun.max.fill"
    case calendar = "calendar"
    case info = "info.circle.fill"

    case paperplane = "paperplane"

    case trash = "trash"
}

enum TripSeasonIcon: String, Setting.Icon {
    case summer = "sun.max.fill"
    case autumn = "cloud.fill"
    case winter = "snowflake"
    case spring = "leaf.fill"
}

public struct TripInfoView: View {

    @State private var viewModel: TripInfoViewModel
    @State private var isDescriptionCollapsed = true

    @State private var isSeasonPickerPresented = false
    @State private var isCategoryPickerPresented = false

    @State private var isStartDatePickerPresented = false
    @State private var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0

    @State private var imageEditingDialogShown = false
    @State private var photoPickerShown = false

    @Environment(\.appRouter) private var router

    var tripSeasonIcon: TripSeasonIcon {
        switch viewModel.trip.season {
        case .summer: return .summer
        case .autumn: return .autumn
        case .winter: return .winter
        case .spring: return .spring
        }
    }

    var datesSettingValue: String {
        if let startDate = viewModel.trip.startDate, let endDate = viewModel.trip.endDate {
            "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            "Не выбрано"
        }
    }

    public init(trip: Trip) {
        viewModel = TripInfoViewModel(trip: trip)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            avatar.padding(.top, 4)

            VStack(spacing: 16) {
                if viewModel.state == .editing { nameEditing }
                if viewModel.state == .default { publishing }
                general
                if viewModel.state == .default {
                    pins
                    description
                    privacy
                } else {
                    descriptionEditing
                    delete
                }
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)

            Spacer()
        }
        .onAppear { viewModel.setRouter(router) }
        .background(PinzUIAsset.background.swiftUIColor)
        .itemsPickerSheet(
            isPresented: $isSeasonPickerPresented,
            items: TripSeason.allCases,
            selection: $viewModel.trip.season,
        )
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: TripCategory.allCases,
            selection: $viewModel.trip.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.trip.category = .custom(value)
            },
        )
        .datePickerSheet(
            isPresented: $isStartDatePickerPresented,
            date: $viewModel.trip.startDate,
            pickerHeight: $datePickerHeight
        )
        .datePickerSheet(
            isPresented: $isEndDatePickerPresented,
            date: $viewModel.trip.endDate,
            pickerHeight: $datePickerHeight
        )
        .confirmationDialog(
            "Выберите действие",
            isPresented: $imageEditingDialogShown,
            titleVisibility: .visible
        ) {
            Button("Выбрать из галереи") {
                photoPickerShown = true
            }
            Button("Удалить фотографию", role: .destructive) { }
        }
        .customImagePicker(show: $photoPickerShown, croppedImage: Binding {
            return viewModel.trip.image
        } set: { newImage in
            viewModel.dispatch(.setImage(newImage))
        })
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.navigate(.back))
                }
            }, centerView: {
                HeaderTitle(
                    viewModel.trip.name,
                    subtitle: "\(viewModel.trip.category.value), \(viewModel.trip.season.value)"
                )
            }, rightView: {
                PinzButton(type: .icon(.pencil), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.changeState)
                }
            })
        case .editing:
            Header {
                PinzButton(type: .text("Отмена")) {
                    viewModel.dispatch(.changeState)
                }
            } rightView: {
                PinzButton(type: .text("Готово")) {
                    viewModel.dispatch(.changeState)
                }
            }
        }
    }

    private var avatar: some View {
        VStack {
            Image(uiImage: viewModel.trip.image ?? PinzUIAsset.avatar.image)
                .resizable()
                .scaledToFill()
                .frame(120)
                .cornerRadius(60)
                .clipped()

            if viewModel.state == .editing {
                Button {
                    imageEditingDialogShown = true
                } label: {
                    Text("Изменить фотографию")
                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                }
            }
        }
    }

    @ViewBuilder
    private var defaultSettings: some View {
        general
        pins
        description
        privacy
        publishing
    }

    private var privacy: some View {
        PrivacySection(members: viewModel.trip.members)
    }

    private var description: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 0) {
                SettingTitle("Описание")
                if let _ = viewModel.trip.description {
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

            if let description = viewModel.trip.description {
                VStack(spacing: 0) {
                    Text(description)
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
                        leading: .iconTitle(TripInfoIcon.text, "Добавить описание"),
                        trailing: .icon(TripInfoIcon.chevronRight),
                        action: .plain {
                            viewModel.dispatch(.changeState)
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
                    leading: .iconTitle(TripInfoIcon.pins, "Пины путешествия"),
                    trailing: .icon(TripInfoIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.pinsList)) }
                )),
            ],
        )
    }

    @ViewBuilder
    private var general: some View {
        let defaultSettings: [Setting] = [
            .default(Setting.DefaultSetting(
                id: "tripSeason",
                leading: .iconTitle(tripSeasonIcon, "Сезон"),
                trailing: .valuesIcon([.text(viewModel.trip.season.value)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripCategory",
                leading: .iconTitle(TripInfoIcon.info, "Категория"),
                trailing: .valuesIcon([.text(viewModel.trip.category.value)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripDates",
                leading: .iconTitle(TripInfoIcon.calendar, "Даты"),
                trailing: .valuesIcon([.text(datesSettingValue)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "tripSeasonPicker",
                leading: .iconTitle(tripSeasonIcon, viewModel.trip.season.value),
                isPickerPresented: $isSeasonPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripCategoryPicker",
                leading: .iconTitle(TripInfoIcon.info, viewModel.trip.category.value),
                isPickerPresented: $isCategoryPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripStartDatePicker",
                leading: .iconTitle(TripInfoIcon.calendar, "Дата начала"),
                value: .text(viewModel.trip.startDate?.formattedToDayMonthYear ?? "Не выбрано"),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripEndDatePicker",
                leading: .iconTitle(TripInfoIcon.calendar, "Дата конца"),
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
                    leading: .iconTitle(TripInfoIcon.paperplane, "Опубликовать путешествие"),
                    trailing: .icon(TripInfoIcon.chevronRight),
                    action: .plain { viewModel.dispatch(.navigate(.selectPins)) }
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
                    placeholder: "Название путешествия"
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
                    text: Binding(get: {
                        viewModel.trip.description ?? ""
                    }, set: { value in
                        viewModel.trip.description = value
                    }),
                    placeholder: "Описание путешествия",
                    style: .multiline
                ))
            ])
        }
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "tripDelete",
                leading: .iconTitle(TripInfoIcon.trash, "Удалить путешествие"),
                trailing: .icon(TripInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
