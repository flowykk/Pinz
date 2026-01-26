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
    case none = "questionmark.circle.fill"
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

    @Environment(\.appRouter) private var router

    var tripSeasonIcon: TripSeasonIcon {
        switch viewModel.trip.season {
        case .none: return .none
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
        VStack(spacing: 0) {
            header

            ScrollView {
                avatar.padding(.top, 12)

                VStack(spacing: 16) {
                    if viewModel.state == .editing { nameEditing }
                    general
                    if viewModel.state == .default {
                        pins
                        description
                        privacy
                        publishing
                    } else {
                        descriptionEditing
                        delete
                    }
                }
                .padding(.top, 8)
                .padding(.horizontal, 12)

                Spacer()
            }.scrollIndicators(.hidden)
        }
        .onAppear { viewModel.setRouter(router) }
        .background(PinzUIAsset.background.swiftUIColor)
        .itemsPickerSheet(
            isPresented: $isSeasonPickerPresented,
            items: TripSeason.allCases,
            selection: $viewModel.trip.season
        )
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: TripCategory.allCases,
            selection: $viewModel.trip.category
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
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.back)
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
        PrivacySection(
            members: [
                TripMember(isPrivate: true, username: "flowykk"),
                TripMember(isPrivate: false, username: "kostik"),
            ]
        )
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
                        title: "Добавить описание",
                        icon: TripInfoIcon.text,
                        trailIcon: TripInfoIcon.chevronRight,
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
                title: "Удалить путешествие",
                icon: TripInfoIcon.trash,
                trailIcon: TripInfoIcon.chevronRight,
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
