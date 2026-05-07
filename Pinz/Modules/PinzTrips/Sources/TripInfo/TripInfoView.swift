import SwiftUI
import PinzUI
import PinzDomain
import PinzBase
import PinzAccessibility

enum TripInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case stories = "rectangle.portrait.on.rectangle.portrait.angled"
    case battle = "flame.fill"

    case pins = "pin"

    case text = "text.alignleft"

    case sun = "sun.max.fill"
    case calendar = "calendar"
    case info = "info.circle.fill"
    case person = "person.2"
    case media = "photo.stack"
    case handThumbsdown = "hand.thumbsdown"
    case handThumbsup = "hand.thumbsup"

    case bell = "bell.badge"

    case paperplane = "paperplane"

    case trash = "trash"
    case door = "door.right.hand.open"
}

public struct TripInfoView: View {

    @State private var viewModel: TripInfoViewModel

    @State private var isSeasonPickerPresented = false
    @State private var isCategoryPickerPresented = false

    @State private var isStartDatePickerPresented = false
    @State private var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0

    @State private var imageEditingDialogShown = false
    @State private var photoPickerShown = false

    @State private var showDeleteTripAlert = false
    @State private var showLeaveTripAlert = false
    @State private var isStoriesPresented = false

    @State private var areNotificationsEnabled = false

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    var datesSettingValue: String {
        if let startDate = viewModel.trip.startDate, let endDate = viewModel.trip.endDate {
            "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            PinzBaseStrings.Common.Label.notSelected
        }
    }

    public init(trip: Trip, onTripUpdated: (() -> Void)? = nil) {
        viewModel = TripInfoViewModel(trip: trip, onTripUpdated: onTripUpdated)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            avatar.padding(.top, 4)

            VStack(spacing: 12) {
                if viewModel.state == .editing { nameEditing }
                if viewModel.state == .default {
                    publishing
                    pins
                }
                general
                if viewModel.state == .default {
                    notifications
                    tripStats
                    description
                    privacy
                } else {
                    descriptionEditing
                }
                if viewModel.state == .default {
                    specialModes
                }
                risky
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)

            Spacer()
        }
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
        }
        .onChange(of: areNotificationsEnabled) { oldValue, newValue in
            Task {
                await viewModel.asyncDispatch(.updateNotifications(enabled: newValue)) { _ in
                    areNotificationsEnabled = oldValue
                }
            }
        }
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
        .onChange(of: isStartDatePickerPresented) { _, isPresented in
            if !isPresented, let error = viewModel.validateDates() { showToast(error) }
        }
        .onChange(of: isEndDatePickerPresented) { _, isPresented in
            if !isPresented, let error = viewModel.validateDates() { showToast(error) }
        }
        .fullScreenCover(isPresented: $isStoriesPresented) {
            PinStoryView(pins: viewModel.trip.pins)
        }
        .fullScreenCover(isPresented: $viewModel.isPhotoBattlePresented) {
            if let battleViewModel = viewModel.photoBattleViewModel {
                PhotoBattleView(viewModel: battleViewModel)
            }
        }
        .confirmationDialog(
            PinzBaseStrings.Common.Alert.Title.selectAction,
            isPresented: $imageEditingDialogShown,
            titleVisibility: .visible
        ) {
            Button(PinzBaseStrings.Common.Button.selectFromGallery) {
                photoPickerShown = true
            }
            Button(PinzBaseStrings.Common.Button.deletePhoto, role: .destructive) { }
        }
        .customImagePicker(show: $photoPickerShown, croppedImage: Binding {
            return viewModel.trip.image
        } set: { newImage in
            viewModel.dispatch(.setImage(newImage))
        })
        .alert(PinzBaseStrings.TripInfo.Alert.DeleteTrip.title, isPresented: $showDeleteTripAlert) {
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) {
                showDeleteTripAlert = false
            }
            Button(PinzBaseStrings.TripInfo.Alert.DeleteTrip.confirm, role: .destructive) {
                Task { await viewModel.deleteTrip() }
            }
        } message: {
            Text(PinzBaseStrings.TripInfo.Alert.DeleteTrip.message)
        }
        .alert(PinzBaseStrings.TripInfo.Alert.LeaveTrip.title, isPresented: $showLeaveTripAlert) {
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) {
                showLeaveTripAlert = false
            }
            Button(PinzBaseStrings.TripInfo.Alert.LeaveTrip.confirm, role: .destructive) {
                Task { await viewModel.asyncDispatch(.leaveTrip) }
            }
        } message: {
            Text(PinzBaseStrings.TripInfo.Alert.LeaveTrip.message)
        }
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .default:
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(
                    viewModel.trip.name,
                    subtitle: "\(viewModel.trip.category.value), \(viewModel.trip.season.value)"
                )
                .pinzA11y(.trip(.row(.headerTitleDetail)))
            }, rightView: {
                HStack(spacing: 10) {
                    PinzButton(
                        type: .icon(.pencil),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.changeState) }
                    )
                    .pinzA11y(.trip(.button(.edit)))
                }
            })
        case .editing:
            Header {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.cancel),
                    action: .plain { viewModel.dispatch(.changeState) }
                )
                .pinzA11y(.trip(.button(.cancel)))
            } rightView: {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.done),
                    action: .async {
                        await viewModel.asyncDispatch(.editTrip)
                    }
                )
                .pinzA11y(.trip(.button(.done)))
            }
        }
    }

    private var avatar: some View {
        VStack {
            tripImage
                .frame(120)
                .cornerRadius(60)
                .clipped()

            if let battleHint = viewModel.photoBattleAvailabilityMessage {
                Text(battleHint)
                    .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }

            if viewModel.state == .editing {
                Button {
                    imageEditingDialogShown = true
                } label: {
                    Text(PinzBaseStrings.Common.Button.editPhoto)
                        .roundedFont(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                }
            }
        }
    }

    @ViewBuilder
    private var tripImage: some View {
        if let localImage = viewModel.trip.image {
            image(for: localImage)
        } else if let url = URL(string: viewModel.trip.coverUrl ?? "") {
            LoadableImageThumbnail(url: url) { state in
                remoteTripImage(for: state)
            }
        } else {
            image(for: PinzDomainAsset.groupPlaceholder.image)
        }
    }

    @ViewBuilder
    private func remoteTripImage(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .overlay {
                    ProgressView()
                        .tint(.white)
                }
        case .ready(let readyImage):
            image(for: readyImage)
        case .failure:
            image(for: PinzDomainAsset.groupPlaceholder.image)
        }
    }

    private func image(for uiImage: UIImage) -> some View {
        Image(uiImage: uiImage)
            .resizable()
            .scaledToFill()
            .frame(120)
            .cornerRadius(60)
            .clipped()
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
            initialSelection: PrivacyIcon.from(isPrivate: viewModel.trip.privacyLevel?.lowercased() == "private"),
            onSelectionChanged: { [viewModel] in viewModel.dispatch(.updatePrivacy($0)) }
        )
    }

    private var description: some View {
        DescriptionView(
            description: viewModel.trip.description,
            contentAccessibilityIdentifier: PinzElement.trip(.row(.description)).accessibilityID,
            onAddAction: {
                viewModel.dispatch(.changeState)
            }
        )
    }

    private var pins: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "tripPins",
                    leading: .iconTitle(TripInfoIcon.pins, PinzBaseStrings.TripInfo.Label.pins),
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
                leading: .iconTitle(TripSeason.icon(for: viewModel.trip.season), PinzBaseStrings.TripInfo.Label.season),
                trailing: .valuesIcon([.text(viewModel.trip.season.value)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripCategory",
                leading: .iconTitle(TripInfoIcon.info, PinzBaseStrings.TripInfo.Label.category),
                trailing: .valuesIcon([.text(viewModel.trip.category.value)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
            .default(Setting.DefaultSetting(
                id: "tripDates",
                leading: .iconTitle(TripInfoIcon.calendar, PinzBaseStrings.TripInfo.Label.dates),
                trailing: .valuesIcon([.text(datesSettingValue)], TripInfoIcon.chevronRight),
                action: .plain { viewModel.dispatch(.changeState) }
            )),
        ]

        let editingSettings: [Setting] = [
            .picker(Setting.PickerSetting(
                id: "tripSeasonPicker",
                leading: .iconTitle(TripSeason.icon(for: viewModel.trip.season), viewModel.trip.season.value),
                isPickerPresented: $isSeasonPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripCategoryPicker",
                leading: .iconTitle(TripInfoIcon.info, viewModel.trip.category.value),
                isPickerPresented: $isCategoryPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripStartDatePicker",
                leading: .iconTitle(TripInfoIcon.calendar, PinzBaseStrings.Common.Label.startDate),
                value: .text(viewModel.trip.startDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                isPickerPresented: $isStartDatePickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "tripEndDatePicker",
                leading: .iconTitle(TripInfoIcon.calendar, PinzBaseStrings.Common.Label.endDate),
                value: .text(viewModel.trip.endDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                isPickerPresented: $isEndDatePickerPresented
            )),
        ]

        SettingsGroup(
            title: PinzBaseStrings.TripInfo.Header.general,
            settings: viewModel.state == .default ? defaultSettings : editingSettings
        ).animation(.default, value: viewModel.trip)
    }

    private var tripStats: some View {
        SettingsGroup(
            settings: [
                .default(Setting.DefaultSetting(
                    id: "tripMembers",
                    leading: .iconTitle(TripInfoIcon.person, PinzBaseStrings.TripMembers.Title.main),
                    trailing: .values([.text(String(viewModel.trip.participantsCount))])
                )),
                .default(Setting.DefaultSetting(
                    id: "tripMedia",
                    leading: .iconTitle(TripInfoIcon.media, PinzBaseStrings.Statistics.Label.media),
                    trailing: .values([.text(String(viewModel.trip.mediaCount))])
                )),
            ],
        )
    }

    private var notifications: some View {
        SettingsGroup(
            settings: [
                .toggle(Setting.ToggleSetting(
                    id: "tripNotifications",
                    leading: .iconTitle(TripInfoIcon.bell, PinzBaseStrings.TripInfo.Label.notifications),
                    value: $areNotificationsEnabled
                )),
            ],
        )
    }

    private var specialModes: some View {
        var items: [Setting] = []

        if !viewModel.trip.pins.isEmpty {
            items.append(.default(Setting.DefaultSetting(
                id: "tripStories",
                leading: .iconTitle(TripInfoIcon.stories, PinzBaseStrings.TripInfo.Button.stories),
                trailing: .icon(TripInfoIcon.chevronRight),
                action: .plain { isStoriesPresented = true }
            )))
        }

        items.append(.default(Setting.DefaultSetting(
            id: "tripPhotoBattle",
            leading: .iconTitle(
                TripInfoIcon.battle,
                Setting.Title(
                    title: PinzBaseStrings.TripInfo.Button.photoBattle,
                    subtitle: viewModel.canStartPhotoBattle ? nil : viewModel.photoBattleAvailabilityMessage
                )
            ),
            trailing: .icon(TripInfoIcon.chevronRight),
            action: (viewModel.canStartPhotoBattle && !viewModel.isStartingBattle && !viewModel.isPhotoBattlePresented)
            ? .async { await viewModel.startPhotoBattle() }
            : nil
        )))

        return SettingsGroup(
            title: PinzBaseStrings.TripInfo.Section.specialModes,
            settings: items
        )
    }

    @ViewBuilder
    private var publishing: some View {
        if viewModel.trip.isPublished {
            SettingsGroup(
                title: PinzBaseStrings.TripInfo.Header.public,
                settings: [
                    .default(Setting.DefaultSetting(
                        id: "tripLikes",
                        leading: .iconTitle(TripInfoIcon.handThumbsup, PinzBaseStrings.Statistics.Label.likes),
                        trailing: .values([.text(String(viewModel.trip.likesCount))])
                    )),
                    .default(Setting.DefaultSetting(
                        id: "tripDislikes",
                        leading: .iconTitle(TripInfoIcon.handThumbsdown, PinzBaseStrings.Statistics.Label.dislikes),
                        trailing: .values([.text(String(viewModel.trip.dislikesCount))])
                    )),
                ],
                subtitle: nil
            )
        } else {
            SettingsGroup(
                title: PinzBaseStrings.TripInfo.Header.public,
                settings: [
                    .default(Setting.DefaultSetting(
                        id: "tripPublishing",
                        leading: .iconTitle(TripInfoIcon.paperplane, PinzBaseStrings.TripInfo.Label.publishTrip),
                        trailing: .icon(TripInfoIcon.chevronRight),
                        action: .plain { viewModel.dispatch(.navigate(.selectPins)) }
                    )),
                ],
                subtitle: nil
            )
        }
    }

    private var nameEditing: some View {
        SettingsGroup(
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "tripNameTextField",
                    text: $viewModel.trip.name,
                    placeholder: PinzBaseStrings.TripInfo.Placeholder.name
                )),
            ],
            subtitle: PinzBaseStrings.TripInfo.Hint.tripNameRules
        )
    }

    private var descriptionEditing: some View {
        DescriptionEditingView(
            title: PinzBaseStrings.Common.Label.description,
            subtitle: PinzBaseStrings.TripInfo.Hint.descriptionLimit,
            text: Binding(get: {
                viewModel.trip.description ?? ""
            }, set: { value in
                viewModel.trip.description = value
            }),
            placeholder: PinzBaseStrings.TripInfo.Placeholder.description,
            textFieldId: "tripDescriptionEditingTextField"
        )
    }

    private var risky: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "tripLeave",
                leading: .iconTitle(TripInfoIcon.door, PinzBaseStrings.TripInfo.Button.leave),
                trailing: .icon(TripInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { showLeaveTripAlert = true }
            )),
            .default(Setting.DefaultSetting(
                id: "tripDelete",
                leading: .iconTitle(TripInfoIcon.trash, PinzBaseStrings.TripInfo.Button.delete),
                trailing: .icon(TripInfoIcon.chevronRight),
                style: .destructive,
                action: .plain {
                    showDeleteTripAlert = true
                }
            ))
        ])
    }
}
