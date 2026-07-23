import SwiftUI
import PinzUI
import PinzBase
import PinzDomain

private enum PublicProfileIcon: String, Setting.Icon {
    case heart = "heart"
    case chevronRight = "chevron.right"
}

public struct PublicProfileView: View {

    @State private var viewModel: PublicProfileViewModel

    @Environment(\.appRouter) private var router

    public init(userId: String) {
        viewModel = PublicProfileViewModel(userId: userId)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                if !viewModel.isLoading {
                    VStack(spacing: 12) {
                        avatarSection
                            .padding(.top, 12)
                        
                        SettingsGroup(settings: [
                            .default(Setting.DefaultSetting(
                                id: "publicProfileWishlist",
                                leading: .iconTitle(PublicProfileIcon.heart, PinzBaseStrings.Profile.Label.wishlist),
                                trailing: .icon(PublicProfileIcon.chevronRight),
                                action: .plain { viewModel.dispatch(.navigate(.wishlist)) }
                            ))
                        ])
                        .padding(.horizontal, 12)

                        Spacer()
                    }
                    .padding(.bottom, 24)
                }
            }

            if viewModel.isLoading {
                LoadingView()
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            Task { try? await viewModel.asyncDispatch(.loadProfile) }
        }
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(viewModel.username)
        })
    }

    private var avatarSection: some View {
        VStack(spacing: 4) {
            avatarImage
            Text(viewModel.username)
                .roundedFont(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
        }
    }

    @ViewBuilder
    private var avatarImage: some View {
        if let urlString = viewModel.avatarUrl, let url = URL(string: urlString) {
            LoadableImageThumbnail(url: url) { state in
                remoteAvatarImage(for: state)
            }
        } else {
            placeholderAvatar(for: ImageProviderType.user.placeholder)
        }
    }

    @ViewBuilder
    private func remoteAvatarImage(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .frame(120)
                .cornerRadius(60)
                .overlay { ProgressView().tint(.white) }
                .clipped()
        case .ready(let image):
            placeholderAvatar(for: image)
        case .failure:
            placeholderAvatar(for: ImageProviderType.user.placeholder)
        }
    }

    private func placeholderAvatar(for image: UIImage) -> some View {
        Image(uiImage: image)
            .resizable()
            .scaledToFill()
            .frame(120)
            .cornerRadius(60)
            .clipped()
    }
}
