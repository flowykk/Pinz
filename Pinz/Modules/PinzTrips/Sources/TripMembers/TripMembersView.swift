import SwiftUI
import PinzUI
import PinzBase

public struct TripMembersView: View {

    @State private var viewModel: TripMembersViewModel

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = TripMembersViewModel()
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
                            leading: .imageTitle(member.avatar, member.username)
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
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.TripMembers.Title.main)
        })
    }
}
