import SwiftUI
import PinzUI
import PinzDomain

private enum FeedFilterIcon: String, Setting.Icon {
    case checkmark = "checkmark"
}

struct FeedFilterView: View {

    @State private var draft: FeedFilterModel

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

    var body: some View {
        ZStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    Text("Фильтры")
                        .roundedFont(size: 24, weight: .bold)
                        .padding(.horizontal, 16)

                    SettingsGroup(title: "Категория", settings: categorySettings)
                        .padding(.horizontal, 12)

                    SettingsGroup(title: "Сезон", settings: seasonSettings)
                        .padding(.horizontal, 12)

                    SettingsGroup(
                        title: "Локация",
                        settings: locationSettings,
                        subtitle: (!draft.city.isEmpty && !draft.country.isEmpty)
                            ? "Город имеет приоритет над страной"
                            : nil
                    )
                    .padding(.horizontal, 12)

                    SettingsGroup(title: "Сортировка", settings: sortSettings)
                        .padding(.horizontal, 12)

                    Spacer(minLength: 150)
                }
                .padding(.top, 20)
            }

            BottomGradientWithButtons {
                HStack(spacing: 6) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Сбросить"),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { onReset() }
                    )
                    PinzButton(
                        type: .slot(style: .primary, title: "Применить"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { onApply(draft) }
                    )
                }
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    private var categorySettings: [Setting] {
        var rows: [Setting] = [
            selectionRow(id: "cat_none", label: "Не выбрано", selected: draft.category == .none) {
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
            selectionRow(id: "sea_none", label: "Не выбрано", selected: draft.season == .none) {
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

    private var locationSettings: [Setting] {
        [
            .textField(Setting.TextFieldSetting(
                id: "city",
                text: $draft.city,
                placeholder: "Город"
            )),
            .textField(Setting.TextFieldSetting(
                id: "country",
                text: $draft.country,
                placeholder: "Страна"
            ))
        ]
    }

    private var sortSettings: [Setting] {
        var rows: [Setting] = [
            selectionRow(id: "sort_none", label: "По умолчанию", selected: draft.sortBy == nil) {
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
