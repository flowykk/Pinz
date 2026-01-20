import SwiftUI
import PinzUI
import PinzDomain

enum TripInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case text = "text.alignleft"

    case sun = "sun.max.fill"
    case calendar = "calendar"
    case info = "info.circle.fill"

    case paperplane = "paperplane"
}

public struct TripInfoView: View {

    @State private var viewModel: TripInfoViewModel
    @State private var isDescriptionCollapsed = true
    @State private var isSeasonPickerPresented = false
    @State private var isCategoryPickerPresented = false
    @Environment(\.appRouter) private var router

    var datesSettingValue: String {
        if let startDate = viewModel.trip.startDate, let endDate = viewModel.trip.endDate {
            "\(startDate) — \(endDate)"
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

                VStack(spacing: 18) {
                    switch viewModel.state {
                    case .default:
                        defaultSettings
                    case .editing:
                        editingSettings
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
                HeaderTitle(viewModel.trip.name)
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
        privacy
        general
        description
        publishing
    }

    @ViewBuilder
    private var editingSettings: some View {
        nameEditing
        general
        descriptionEditing
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

    @ViewBuilder
    private var general: some View {
        let defaultSettings: [Setting] = [
            .default(Setting.DefaultSetting(
                id: "tripSeason",
                title: "Сезон",
                icon: TripInfoIcon.sun,
                values: [.text(viewModel.trip.season.value)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain { }
            )),
            .default(Setting.DefaultSetting(
                id: "tripCategory",
                title: "Категория",
                icon: TripInfoIcon.info,
                values: [.text(viewModel.trip.category.value)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain {}
            )),
            .default(Setting.DefaultSetting(
                id: "tripDates",
                title: "Даты",
                icon: TripInfoIcon.calendar,
                values: [.text(datesSettingValue)],
                trailIcon: TripInfoIcon.chevronRight,
                action: .plain {}
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "tripSeasonPicker",
                items: TripSeason.allCases,
                title: viewModel.trip.season.value,
                value: $viewModel.trip.season,
                icon: TripInfoIcon.sun,
                action: .plain { isSeasonPickerPresented = true }
            )),
            .picker(Setting.PickerSetting(
                id: "tripCategoryPicker",
                items: TripCategory.allCases,
                title: viewModel.trip.category.value,
                value: $viewModel.trip.category,
                icon: TripInfoIcon.info,
                action: .plain { isCategoryPickerPresented = true }
            )),
        ]

        SettingsGroup(
            title: "Общая информация",
            settings: viewModel.state == .default ? defaultSettings : editingSettings
        )
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
}
