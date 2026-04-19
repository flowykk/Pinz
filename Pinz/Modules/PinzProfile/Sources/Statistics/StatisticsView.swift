import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

private enum StatisticsIcon: String, Setting.Icon {
    case globe = "globe.americas.fill"
    case pin = "pin"
    case media = "photo"
    case handThumbsdown = "hand.thumbsdown"
    case handThumbsup = "hand.thumbsup"
    case shield = "shield"
    case city = "building.2"
    case flag = "flag"
}

public struct StatisticsView: View {

    @State private var viewModel: StatisticsViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = StatisticsViewModel()
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                }, centerView: {
                    HeaderTitle(PinzBaseStrings.Statistics.Title.main)
                })
            } content: {
                if !viewModel.isLoading {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 12) {
                            SettingsGroup(
                                settings: [
                                    .default(Setting.DefaultSetting(
                                        id: "statsTrips",
                                        leading: .iconTitle(StatisticsIcon.globe, PinzBaseStrings.Statistics.Label.trips),
                                        trailing: .values([.text(String(viewModel.tripsCount))])
                                    )),
                                    .default(Setting.DefaultSetting(
                                        id: "statsPins",
                                        leading: .iconTitle(StatisticsIcon.pin, PinzBaseStrings.Statistics.Label.pins),
                                        trailing: .values([.text(String(viewModel.pinsCount))])
                                    )),
                                    .default(Setting.DefaultSetting(
                                        id: "statsMedia",
                                        leading: .iconTitle(StatisticsIcon.media, PinzBaseStrings.Statistics.Label.media),
                                        trailing: .values([.text(String(viewModel.mediaCount))])
                                    )),
                                    .default(Setting.DefaultSetting(
                                        id: "statsLikes",
                                        leading: .iconTitle(StatisticsIcon.handThumbsup, PinzBaseStrings.Statistics.Label.likes),
                                        trailing: .values([.text(String(viewModel.likesCount))])
                                    )),
                                    .default(Setting.DefaultSetting(
                                        id: "statsDislikes",
                                        leading: .iconTitle(StatisticsIcon.handThumbsdown, PinzBaseStrings.Statistics.Label.dislikes),
                                        trailing: .values([.text(String(viewModel.dislikesCount))])
                                    )),
                                    .default(Setting.DefaultSetting(
                                        id: "statsBattles",
                                        leading: .iconTitle(StatisticsIcon.shield, PinzBaseStrings.Statistics.Label.battles),
                                        trailing: .values([.text(String(viewModel.battlesCount))])
                                    )),
                                ]
                            )

                            locationsGroup(
                                title: PinzBaseStrings.Statistics.Group.countries,
                                icon: .flag,
                                locations: viewModel.visitedCountries
                            )

                            locationsGroup(
                                title: PinzBaseStrings.Statistics.Group.cities,
                                icon: .city,
                                locations: viewModel.visitedCities
                            )
                        }
                        .padding(.horizontal, 12)
                        .padding(.top, 12)
                    }
                }
            }

            if viewModel.isLoading {
                LoadingView()
            }
        }.onAppear {
            viewModel.setRouter(router)
            viewModel.dispatch(.loadStats)
        }
    }

    private func locationSubtitle(_ location: VisitedLocationDTO) -> String? {
        var parts: [String] = []

        if let lastVisitedAtUnix = location.lastVisitedAtUnix {
            let date = Date(timeIntervalSince1970: TimeInterval(lastVisitedAtUnix)).formattedToDayMonthYear
            parts.append(PinzBaseStrings.Statistics.Visited.lastVisited(date))
        }

        if let visitsCount = location.visitsCount {
            parts.append(PinzBaseStrings.Statistics.Visited.visitsCount("\(visitsCount)"))
        }

        guard !parts.isEmpty else {
            return nil
        }

        return parts.joined(separator: " • ")
    }

    @ViewBuilder
    private func locationsGroup(
        title: String,
        icon: StatisticsIcon,
        locations: [VisitedLocationDTO]
    ) -> some View {
        let normalizedLocations: [(location: VisitedLocationDTO, name: String)] = locations.compactMap { location in
            guard let name = location.name?.trimmingCharacters(in: .whitespacesAndNewlines),
                  !name.isEmpty else { return nil }
            return (location: location, name: name)
        }

        if !normalizedLocations.isEmpty {
            SettingsGroup(
                title: title,
                settings: normalizedLocations
                    .enumerated()
                    .map { index, item in
                        Setting.default(
                            Setting.DefaultSetting(
                                id: "statistics.visited.locations.\(icon.rawValue).\(index)",
                                leading: .iconTitle(
                                    icon,
                                    Setting.Title(
                                        title: item.name,
                                        subtitle: locationSubtitle(item.location)
                                    )
                                )
                            )
                        )
                    }
            )
        } else {
            SettingsGroup(
                title: title,
                settings: [
                    .default(
                        Setting.DefaultSetting(
                            id: "statistics.visited.locations.empty.\(title)",
                            leading: Setting.Leading.title(
                                Setting.Title(title: PinzBaseStrings.Common.Label.notSelected, subtitle: nil)
                            ),
                            trailing: nil
                        )
                    )
                ]
            )
        }
    }
}
