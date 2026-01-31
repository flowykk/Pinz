import SwiftUI
import PinzDomain

enum PrivacyIcon: String, Setting.Icon, SegmentedItem {
    case lockOpened = "lock.fill"
    case lockClosed = "lock.open.fill"

    var id: Self { self }

    var content: SegmentedItemContent {
        switch self {
        case .lockOpened: .icon(rawValue, PinzUIAsset.accentGreen.swiftUIColor)
        case .lockClosed: .icon(rawValue, PinzUIAsset.accentRed.swiftUIColor)
        }
    }
}

public struct PrivacySection: View {

    let members: [TripMember]

    @State var privacySelection: PrivacyIcon = .lockClosed

    public init(members: [TripMember]) {
        self.members = members
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle("Участники и приватность")
                .padding(.leading, 12)

            SegmentedPicker(selection: $privacySelection, items: [.lockOpened, .lockClosed])

            VStack(spacing: 0) {
                SettingsGroup(settings: members.map { member in
                    .default(Setting.DefaultSetting(
                        id: "privacy\(member.username)",
                        leading: .imageTitle(member.avatar, member.username),
                        trailing: .values([
                            member.isPrivate
                                ? .icon(PrivacyIcon.lockClosed, PinzUIAsset.accentRed.swiftUIColor)
                                : .icon(PrivacyIcon.lockOpened, PinzUIAsset.accentGreen.swiftUIColor)
                        ])
                    ))
                })
            }
            .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
            .cornerRadius(26)

            SettingSubtitle(
                privacySelection == .lockClosed
                    ? "Путешествие не будет отображаться в общей ленте"
                    : "Путешествие будет отображаться в общей ленте"
            )
            .padding(.top, -2)
            .padding(.leading, 12)
        }
    }

    private func isTotallyPrivate() -> Bool {
        members.allSatisfy(\.isPrivate)
    }
}
