import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

private enum FeedFilterIcon: String, Setting.Icon {
    case checkmark = "checkmark"
}

struct FeedFilterView: View {

    @State private var draft: FeedFilterModel
    @State private var isCityPickerPresented = false
    @State private var isCountryPickerPresented = false

    let onApply: (FeedFilterModel) -> Void
    let onReset: () -> Void

    init(
        currentFilters: FeedFilterModel,
        onApply: @escaping (FeedFilterModel) -> Void,
        onReset: @escaping () -> Void
    ) {
        _draft = State(initialValue: currentFilters)
        self.onApply = onApply
        self.onReset = onReset
    }

    private var citySelectionBinding: Binding<String> {
        Binding(
            get: { draft.city },
            set: { newValue in
                draft.city = newValue
                if !newValue.isEmpty {
                    draft.country = ""
                }
            }
        )
    }

    private var countrySelectionBinding: Binding<String> {
        Binding(
            get: { draft.country },
            set: { newValue in
                draft.country = newValue
                if !newValue.isEmpty {
                    draft.city = ""
                }
            }
        )
    }

    var body: some View {
        ZStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    Text(PinzBaseStrings.Feed.Filter.title)
                        .roundedFont(size: 24, weight: .bold)
                        .padding(.horizontal, 16)

                    SettingsGroup(
                        title: PinzBaseStrings.Feed.Filter.Group.location,
                        settings: locationSettings,
                        subtitle: nil
                    )
                    .padding(.horizontal, 12)

                    SettingsGroup(title: PinzBaseStrings.Feed.Filter.Group.category, settings: categorySettings)
                        .padding(.horizontal, 12)

                    SettingsGroup(title: PinzBaseStrings.Feed.Filter.Group.season, settings: seasonSettings)
                        .padding(.horizontal, 12)

                    SettingsGroup(title: PinzBaseStrings.Feed.Filter.Group.sort, settings: sortSettings)
                        .padding(.horizontal, 12)

                    Spacer(minLength: 150)
                }
                .padding(.top, 20)
            }

            BottomGradientWithButtons {
                HStack(spacing: 6) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.Common.Button.reset),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { onReset() }
                    )
                    PinzButton(
                        type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.apply),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { onApply(draft) }
                    )
                }
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .sheet(isPresented: $isCityPickerPresented) {
            FeedGeoPickerSheet(
                title: PinzBaseStrings.Feed.Filter.Label.city,
                segment: .cities,
                selectedSlug: citySelectionBinding,
                isPresented: $isCityPickerPresented
            )
            .pinzSheet()
        }
        .sheet(isPresented: $isCountryPickerPresented) {
            FeedGeoPickerSheet(
                title: PinzBaseStrings.Feed.Filter.Label.country,
                segment: .countries,
                selectedSlug: countrySelectionBinding,
                isPresented: $isCountryPickerPresented
            )
            .pinzSheet()
        }
    }

    private var locationSettings: [Setting] {
        let cityValue: Setting.Value = .text(
            draft.city.isEmpty
                ? PinzBaseStrings.Common.Label.notSelected
                : FeedGeoCatalog.cityDisplay(forSlug: draft.city)
        )
        let countryValue: Setting.Value = .text(
            draft.country.isEmpty
                ? PinzBaseStrings.Common.Label.notSelected
                : FeedGeoCatalog.countryDisplay(forSlug: draft.country)
        )
        return [
            .picker(Setting.PickerSetting(
                id: "city",
                leading: .title(PinzBaseStrings.Feed.Filter.Label.city),
                value: cityValue,
                isPickerPresented: $isCityPickerPresented
            )),
            .picker(Setting.PickerSetting(
                id: "country",
                leading: .title(PinzBaseStrings.Feed.Filter.Label.country),
                value: countryValue,
                isPickerPresented: $isCountryPickerPresented
            ))
        ]
    }

    private var categorySettings: [Setting] {
        var rows: [Setting] = [
            selectionRow(id: "cat_none", label: PinzBaseStrings.Common.Label.notSelected, selected: draft.category == .none) {
                draft.category = .none
            }
        ]
        TripCategory.allCases.forEach { cat in
            rows.append(selectionRow(id: "cat_\(cat.value)", label: cat.value, selected: draft.category == cat) {
                draft.category = cat
            })
        }
        return rows
    }

    private var seasonSettings: [Setting] {
        var rows: [Setting] = [
            selectionRow(id: "sea_none", label: PinzBaseStrings.Common.Label.notSelected, selected: draft.season == .none) {
                draft.season = .none
            }
        ]
        TripSeason.allCases.forEach { s in
            rows.append(selectionRow(id: "sea_\(s.value)", label: s.value, selected: draft.season == s) {
                draft.season = s
            })
        }
        return rows
    }

    private var sortSettings: [Setting] {
        var rows: [Setting] = [
            selectionRow(id: "sort_none", label: PinzBaseStrings.Feed.Filter.Sort.default, selected: draft.sortBy == nil) {
                draft.sortBy = nil
            }
        ]
        FeedSortBy.allCases.forEach { sort in
            rows.append(selectionRow(id: "sort_\(sort.rawValue)", label: sort.displayName, selected: draft.sortBy == sort) {
                draft.sortBy = sort
            })
        }
        return rows
    }

    private func selectionRow(
        id: String,
        label: String,
        selected: Bool,
        action: @escaping () -> Void
    ) -> Setting {
        .default(Setting.DefaultSetting(
            id: id,
            leading: .title(label),
            trailing: selected ? .icon(FeedFilterIcon.checkmark, PinzUIAsset.accentGreen.swiftUIColor) : nil,
            action: .plain(action)
        ))
    }
}
