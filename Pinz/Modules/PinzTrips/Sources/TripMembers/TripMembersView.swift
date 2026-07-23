import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

private enum TripMemberIcon: String, Setting.Icon {
    case adminStar = "star.fill"
    case chevronRight = "chevron.right"
}

public struct TripMembersView: View {

    @State private var viewModel: TripMembersViewModel
    @State private var isInvitePresented = false

    @Environment(\.appRouter) private var router

    private let tripId: String

    public init(tripId: String, participants: [TripParticipantDTO], currentUserId: String?) {
        self.tripId = tripId
        viewModel = TripMembersViewModel(participants: participants, currentUserId: currentUserId)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            VStack(spacing: 8) {
                SettingsGroup(settings: [
                    .textField(Setting.TextFieldSetting(
                        id: "memberSearch",
                        text: $viewModel.searchText,
                        placeholder: PinzBaseStrings.TripMembers.Placeholder.search
                    ))
                ])

                if viewModel.isLoading {
                    LoadingView()
                } else {
                    SettingsGroup(settings: viewModel.filteredMembers.map { member in
                        .default(Setting.DefaultSetting(
                            id: member.id,
                            leading: .imageTitle(member.avatar, member.username),
                            trailing: trailing(for: member),
                            action: isSelf(member) ? nil : .plain { viewModel.dispatch(.openProfile(member)) }
                        ))
                    }).animation(.default, value: viewModel.filteredMembers)
                }
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)

            Spacer()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .fullScreenCover(isPresented: $isInvitePresented) {
            TripInviteView(tripId: tripId)
        }
    }

    private func isSelf(_ member: TripMember) -> Bool {
        member.id == viewModel.currentUserId
    }

    private func trailing(for member: TripMember) -> Setting.Trailing? {
        let isAdmin = member.role == .admin
        if isSelf(member) {
            return isAdmin ? .icon(TripMemberIcon.adminStar) : nil
        } else {
            return isAdmin
                ? .valuesIcon([.icon(TripMemberIcon.adminStar, .yellow)], TripMemberIcon.chevronRight)
                : .icon(TripMemberIcon.chevronRight)
        }
    }

    private var header: some View {
        Header(
            leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            },
            centerView: {
                HeaderTitle(PinzBaseStrings.TripMembers.Title.main)
            },
            rightView: {
                PinzButton(
                    type: .icon(.personAdd),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { isInvitePresented = true }
                )
                .disabledWithOpacity(viewModel.isLoading)
            }
        )
    }
}
